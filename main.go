package main

import (
	"flag"
	xmpp "github.com/3M3RY/go-xmpp"
	"net/textproto"
	"os"
)

var (
	childLimit        = flag.Int("n", 8, "limit concurrent processing routines to n")
	hostname          = flag.String("hostname", "", "hostname to report to clients, defaults to $HOSTNAME")
	twoTwentyGreeting string
	twoFiftyReply     string
)

func isValid(recipient string) bool {
	return true
}

func newXMPPClient() {

}

func process(conn *net.Conn) {
	defer c.Close()
	w := textproto.NewWriter(bufio.NewWriter(conn))
	err := w.PrintfLine(twoTwentyGreeting)
	if err != nil {
		fmt.Println(err)
		return
	}

	r := textproto.NewReader(bufio.NewReader(conn))
	s, err := r.ReadLine()
	switch s[:4] {
	case "EHLO":
		w.PrintfLine(twoFiftyReply)

	case "HELO":

	default:
		return
	}

	s, err := r.ReadLine()
	if err != nil {
		fmt.Println(err)
		return
	}
	if s[:10] != "MAIL FROM:" {
		fmt.Println("Did not receive proper MAIL command")
		return
	}
	w.PrintfLine("250 OK")

	s, err = r.ReadLine()
	if err != nil {
		fmt.Println(err)
		return
	}
	// TODO may get mail for more than one recipient
	if s[:8] != "RCPT TO:" {
		fmt.Println("Did not recieve RCPT command")
	}

	rcpt := s[8:]
	if !isValid(rcpt) {
		fmt.Println("Ignoring mail for", rcpt)
		return
	}

	s, err := r.ReadLine()
	if err != nil {
		fmt.Println(err)
		return
	}
	if s != "DATA" {
		fmt.Println("Did not receive data command the way we like it")
		return
	}
	w.PrintfLine("354")

	dr := textproto.NewDotReader(conn)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(dr)
	if err != nil {
		fmt.Println("Error: err")
		return
	}

	// TODO just drop everything after the first >
	rcpt = rcpt[1 : len(rcpt)-1]

	xmppMu.Lock()
	defer xmppMu.Unlock()
	if xmppClient == nil {
		xmppClient, err := xmpp.NewClientNoTLS(domain, user, password)
		if err != nil {
			// TODO inform the client that recieving the message has failed
			fmt.Println("Could not connect to XMPP server:", err)
			return
		}
	}
	defer xmppClient.Close()

	err = xmppClient.SendMessage(rcpt, "", buf.String())
	if err != nil {
		// TODO inform the client that recieving the message has failed
		fmt.Println("Error sending XMPP message:", err)
	}
	w.PrintfLine("250 OK")
}

func main() {
	flag.Parse()
	if *hostname == "" {
		&hostname = os.Getenv("HOSTNAME")
	}
	twoTwentyGreeting = fmt.Sprintf("220 %s SMTP to XMPP gateway", *hostname)
	twoFiftyReply = fmt.Sprintf("250-%s", *hostname)

	l, err := net.Listen("tcp", ":25")
	if err != nil {
		fmt.Println("Could not listen on port 25:", err)
		os.Exit(1)
	}
	defer l.CLose()

	childChan := make(chan child, childLimit)
	for i := 0; i < childLimit; i++ {
		newChild(childChan)
	}

	for {
		conn, err := l.Accept()
		process(conn)

		// 450  Requested mail action not taken: mailbox unavailable (e.g.,
		// mailbox busy or temporarily blocked for policy reasons)
		//conn.Close()

	}

}
