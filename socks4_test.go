package socks4_test

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"testing"

	"github.com/bdandy/go-socks4"

	"golang.org/x/net/proxy"
)

var address string

func init() {
	flag.StringVar(&address, "socks4.url", "", "URL of socks4 server to connect to")
}

func TestDial(t *testing.T) {
	flag.Parse()

	addr, err := url.Parse(address)
	if err != nil {
		t.Error(err)
		return
	}

	socks, err := proxy.FromURL(addr, proxy.Direct)
	if err != nil {
		t.Error(err)
		return
	}

	c, err := socks.Dial("tcp", "google.com:80")
	if err != nil {
		if socksErr, ok := err.(socks4.Error); ok {
			t.Fatal(socksErr)
		}
		t.Fatal("unknown error", err)
	}

	defer c.Close()

	_, err = c.Write([]byte("GET /\n"))
	if err != nil {
		t.Error(err)
	}

	buf := bufio.NewReader(c)
	line, _ := buf.ReadString('\n')
	fmt.Print(line)
}
