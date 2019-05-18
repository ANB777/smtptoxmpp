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
	"github.com/BurntSushi/toml"
	"github.com/coreos/go-systemd/activation"
	xmpp "github.com/ehmry/go-xmpp"
	"io"
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

/*
type inetdConn struct {
	stdin  *os.File
	stdout *os.File
}

func (c *inetdConn) Read(p []byte) (int, error) {
	return c.stdin.Read(p)
}

func (c *inetdConn) Write(p []byte) (int, error) {
	return c.stdout.Write(p)
}

func (c *inetdConn) Close() (err error) {
	err = c.stdin.Close()
	if err != nil {
		return nil
	}
	return c.stdin.Close()
}
*/

var subjectRe = regexp.MustCompile(`Subject: (.*)`)
var msgRe = regexp.MustCompile(`(?ms)[\r\n][\r\n]+(.*)$`)

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

func process(conn io.ReadWriteCloser) {
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

	var msgContent string
	if msgs := msgRe.FindStringSubmatch(msg); len(msg) > 1 {
		msgContent = msgs[1]
	}

	for _, recipient := range recipients {
		if smtpAddrRe != nil {
			recipient = smtpAddrRe.ReplaceAllString(recipient, xmppAddrRe)
		}

		err = component.SendMessage(fromAddress, recipient, subject, msgContent)
		if err != nil {
			fmt.Println("XMPP Error: failed to send message: ", err)
			w.PrintfLine("451 Requested action aborted: local error in processing") // see https://www.greenend.org.uk/rjk/tech/smtpreplies.html
			return
		}
	}
	w.PrintfLine("250 OK")
}

var (
	idle              = flag.Duration("idle", time.Minute, "process idle timeout (systemd only)")
	daemonPort        = flag.Uint("port", 0, "run as a persistant server on given port")
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
		fmt.Println("USAGE:", os.Args[0], "[-port PORT_NUMBER] CONFIG_FILE")
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

	component, err = xmpp.NewComponent(config.Xmpp.Domain, config.Xmpp.Name, config.Xmpp.Secret, config.Xmpp.Server, config.Xmpp.Port)
	if err != nil {
		// TODO inform the client that recieving the message has failed
		fmt.Println("XMPP Error: Could not connect to XMPP server, ", err)
		os.Exit(1)
	}
	defer component.Close()
	smtpClients := make(chan net.Conn)

	if *daemonPort != 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *daemonPort))
		if err != nil {
			fmt.Println("SMTP Error:", err)
			os.Exit(1)
		}
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					fmt.Println("SMTP Error: ", err)
					continue
				}
				smtpClients <- conn
			}
		}()
		for {
			conn := <-smtpClients
			process(conn)
		}

	} else {
		listeners, err := activation.Listeners(true)
		if err != nil {
			fmt.Println("SMTP Error: failed to get sockets from environment,", err)
			os.Exit(1)
		}
		for _, l := range listeners {
			defer l.Close()
			go func() {
				for {
					conn, err := l.Accept()
					if err != nil {
						fmt.Println("SMTP Error: ", err)
						continue
					}
					smtpClients <- conn
				}
			}()
		}

		for {
			select {
			case conn := <-smtpClients:
				process(conn)

			case <-time.After(*idle):
				os.Exit(0)
			}
		}
	}
}
