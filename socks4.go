package socks4

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"net/url"
	"strconv"

	typedErrors "github.com/Bogdan-D/go-typed-errors"
	"golang.org/x/net/proxy"
)

const (
	socksVersion = 0x04
	socksConnect = 0x01
	// nolint
	socksBind = 0x02

	accessGranted       = 0x5a
	accessRejected      = 0x5b
	accessIdentRequired = 0x5c
	accessIdentFailed   = 0x5d

	minRequestLen = 8
)

const (
	ErrWrongConnType = typedErrors.String("no support for connections of type")
	ErrDialFailed    = typedErrors.String("socks4 dial")
	ErrConnRejected  = typedErrors.String("connection to remote host was rejected")
	ErrIdentRequired = typedErrors.String("valid ident required")
	ErrIO            = typedErrors.String("i\\o error")
	ErrWrongURL      = typedErrors.String("wrong server url: %s")
	ErrHostUnknown   = typedErrors.String("unable to find IP address of host %s")
	ErrUnknown       = typedErrors.String("unknown socks4 server response %v")
)

var Ident = "nobody@0.0.0.0"

func init() {
	proxy.RegisterDialerType("socks4", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return socks4{url: u, dialer: d}, nil
	})

	proxy.RegisterDialerType("socks4a", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return socks4{url: u, dialer: d}, nil
	})
}

type socks4 struct {
	url    *url.URL
	dialer proxy.Dialer
}

func (s socks4) lookupAddr(host string) (net.IP, error) {
	ip, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return net.IP{}, ErrHostUnknown.WithArgs(host).Wrap(err)
	}

	return ip.IP.To4(), err
}

func (s socks4) prepareRequest(addr string) ([]byte, error) {
	var (
		buf bytes.Buffer
		err error
	)

	host, port, err := s.parseAddr(addr)

	buf.Write([]byte{socksVersion, socksConnect})
	_ = binary.Write(&buf, binary.BigEndian, uint16(port))

	// socks4a defines IP as 0.0.0.x
	var ip = net.IPv4(0, 0, 0, 1)

	if !s.isSocks4a() {
		ip, err = s.lookupAddr(host)
		if err != nil {
			return nil, err
		}
	}

	_ = binary.Write(&buf, binary.BigEndian, ip.To4())
	buf.WriteString(Ident)

	buf.WriteByte(0)

	if s.isSocks4a() {
		buf.WriteString(host)
		buf.WriteByte(0)
	}

	return buf.Bytes(), nil
}

func (s socks4) isSocks4a() bool {
	return s.url.Scheme == "socks4a"
}

func (s socks4) parseAddr(addr string) (host string, iport int, err error) {
	var port string
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return "", 0, ErrWrongURL.WithArgs(addr).Wrap(err)
	}

	iport, err = strconv.Atoi(port)
	if err != nil {
		return "", 0, ErrWrongURL.WithArgs(addr).Wrap(err)
	}

	return
}

func (s socks4) Dial(network, addr string) (c net.Conn, err error) {
	if network != "tcp" && network != "tcp4" {
		return nil, ErrWrongConnType
	}

	c, err = s.dialer.Dial(network, s.url.Host)
	if err != nil {
		return nil, ErrDialFailed.Wrap(err)
	}
	// close connection later if we got an error
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	req, err := s.prepareRequest(addr)
	if err != nil {
		return c, err
	}

	var i int
	i, err = c.Write(req)
	if err != nil {
		return c, ErrIO.Wrap(err)
	} else if i < minRequestLen {
		return c, ErrIO.Wrap(io.ErrShortWrite)
	}

	var resp [8]byte
	i, err = c.Read(resp[:])
	if err != nil && err != io.EOF {
		return c, ErrIO.Wrap(err)
	} else if i != 8 {
		return c, ErrIO.Wrap(io.ErrUnexpectedEOF)
	}

	switch resp[1] {
	case accessGranted:
		return c, nil
	case accessIdentRequired, accessIdentFailed:
		return c, ErrIdentRequired
	case accessRejected:
		return c, ErrConnRejected
	default:
		return c, ErrUnknown.WithArgs(resp[1])
	}
}
