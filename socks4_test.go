package socks4_test

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"testing"

	"github.com/Bogdan-D/go-socks4"

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
	defer c.Close()

	var socksErr socks4.Error
	if err != nil && errors.As(err, &socksErr) {
		switch {
		case socksErr.Equal(socks4.ErrIdentRequired):
		default:
			t.Error(err)
		}
	} else if err != nil {
		t.Error(err)
	}

	_, err = c.Write([]byte("GET /\n"))
	if err != nil {
		t.Error(err)
	}

	buf := bufio.NewReader(c)
	line, err := buf.ReadString('\n')
	fmt.Print(line)
}
