/*
   smtptoxmpp
   Copyright (C) 2013 Emery Hemingway xmpp:emery@fuzzlabs.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/3M3RY/go-systemd/systemd"
	xmpp "github.com/3M3RY/go-xmpp"
	"github.com/BurntSushi/toml"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"
)

type xmppConfig struct {
	Domain string `toml:"domain"`
	Name   string `toml:"name"`
	Secret string `toml:"secret"`
	Server string `toml:"server"`
	Port   int16  `toml:"port"`
	SmtpRe string `toml:"smtpregexp"`
	XmppRe string `toml:"xmppregexp"`
}

type tomlConfig struct {
	Xmpp xmppConfig `toml:"xmpp"`
}

var subjectRe = regexp.MustCompile(`Subject: (.*)`)

func stripAddr(s string) (address string) {
	address = strings.Split(s, "<")[1]
	address = address[:strings.Index(address, ">")]
	return
}

func stripAddrs(s string) (addresses []string) {
	addresses = strings.Split(s, "<")[1:]
	for i, a := range addresses {
		addresses[i] = a[:strings.Index(a, ">")]
	}
	return
}

func process(conn net.Conn) {
	defer conn.Close()
	w := textproto.NewWriter(bufio.NewWriter(conn))
	err := w.PrintfLine(twoTwentyGreeting)
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		return
	}

	r := textproto.NewReader(bufio.NewReader(conn))
	s, err := r.ReadLine()
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		return
	}

	fmt.Println(conn.RemoteAddr(), "\t", s)

	switch s[:4] {
	case "EHLO":
		// I don't know what those extensions are but don't give a shit
		w.PrintfLine(twoFiftyReply + " greets " + s[4:])
		w.PrintfLine("250-8BITMIME ")
		w.PrintfLine("250-SIZE ")
		w.PrintfLine("250-DSN ")
		w.PrintfLine("250 HELP ")
	case "HELO":
		w.PrintfLine(twoFiftyGreeting)
	default:
		fmt.Println("SMTP Error: client sent this shit: ", s)
		return
	}

	s, err = r.ReadLine()
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		return
	}

	if s[:10] != "MAIL FROM:" {
		fmt.Println("SMTP Error: client sent '", s, "' instead of MAIL FROM")
		return
	}
	w.PrintfLine("250 OK")

	s, err = r.ReadLine()
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		fmt.Println(err)
		return
	}
	// TODO may get mail for more than one recipient
	if s[:8] != "RCPT TO:" {
		fmt.Println("SMTP Error: client sent '", s, "' instead of RCPT TO")
	}

	recipients := stripAddrs(s[8:])
	//if !isValid(rcpt) {
	//	fmt.Println("Ignoring mail for", rcpt)
	//	return
	//}
	w.PrintfLine("250 OK")

	s, err = r.ReadLine()
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		return
	}

	for {
		if err != nil {
			fmt.Println("SMTP Error: ", err)
			return
		}
		if s == "DATA" {
			break
		}
		if len(s) > 8 && s[:8] == "RCPT TO:" {
			recipients = append(recipients, stripAddrs(s[8:])...)
			w.PrintfLine("250 OK")
			s, err = r.ReadLine()
			continue
		}
		fmt.Println("SMTP Error: expected DATA, got ", s)
		return
	}
	w.PrintfLine("354 End data with <CR><LF>.<CR><LF>")

	dr := r.DotReader()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(dr)
	if err != nil {
		fmt.Println("SMTP Error: ", err)
		return
	}

	msg := buf.String()

	var subject string
	if subjects := subjectRe.FindStringSubmatch(msg); len(subjects) > 1 {
		subject = subjects[1]
	}

	for _, recipient := range recipients {
		if smtpAddrRe != nil {
			recipient = smtpAddrRe.ReplaceAllString(recipient, xmppAddrRe)
		}

		err = component.SendMessage(fromAddress, recipient, subject, msg)
		if err != nil {
			// TODO inform the client that recieving the message has failed
			fmt.Println("XMPP Error: failed to send message: ", err)
		}
	}
	w.PrintfLine("250 OK")
}

var (
	idle              = flag.Duration("idle", time.Minute, "process idle timeout")
	twoTwentyGreeting string
	twoFiftyGreeting  string
	twoFiftyReply     string
	fromAddress       string
	component         *xmpp.Component
	config            *tomlConfig
	smtpAddrRe        *regexp.Regexp
	xmppAddrRe        string
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("USAGE:", os.Args[0], "CONFIG_FILE")
		os.Exit(1)
	}

	_, err := toml.DecodeFile(args[0], &config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error: could not determine hostname, ", err)
		os.Exit(1)
	}

	fromAddress = config.Xmpp.Name + "." + config.Xmpp.Domain

	twoTwentyGreeting = "220 " + hostname + " SMTP to XMPP gateway"
	twoFiftyGreeting = "250 " + hostname
	twoFiftyReply = "250-" + hostname

	if config.Xmpp.SmtpRe != "" {
		smtpAddrRe = regexp.MustCompile(config.Xmpp.SmtpRe)
		xmppAddrRe = config.Xmpp.XmppRe
	}

	listeners, err := systemd.Listen()
	if err != nil {
		fmt.Println("SMTP Error: failed to get sockets from environment, ", err)
		os.Exit(1)
	}
	smtpClients := make(chan net.Conn)
	for _, l := range listeners {
		go func() {
			defer l.Close()
			for {
				conn, err := l.Accept()
				if err != nil {
					fmt.Println("SMTP Error: ", err)
					continue
				}
				smtpClients <- conn
				// 450  Requested mail action not taken: mailbox unavailable (e.g.,
				// mailbox busy or temporarily blocked for policy reasons)
				//conn.Close()
			}
		}()
	}

	component, err = xmpp.NewComponent(config.Xmpp.Domain, config.Xmpp.Name, config.Xmpp.Secret, config.Xmpp.Server, config.Xmpp.Port)
	if err != nil {
		// TODO inform the client that recieving the message has failed
		fmt.Println("XMPP Error: Could not connect to XMPP server, ", err)
		os.Exit(1)
	}
	defer component.Close()

	for {
		select {
		case conn := <-smtpClients:
			process(conn)

		case <-time.After(*idle):
			os.Exit(0)
		}
	}

}
