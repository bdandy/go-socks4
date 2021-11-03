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
	ErrWrongURL      = typedErrors.String("wrong server url: %s")
	ErrWrongConnType = typedErrors.String("no support for connections of type")
	ErrDialFailed    = typedErrors.String("socks4 server dial error")
	ErrHostUnknown   = typedErrors.String("unable to find IP address of host %s")
	ErrConnRejected  = typedErrors.String("connection to remote host was rejected by socks4 server")
	ErrIdentRequired = typedErrors.String("socks4 server require valid ident: %v")
	ErrSocksServer   = typedErrors.String("socks4 server error")
	ErrUnknown       = typedErrors.String("unknown socks4 server response")
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

func (s socks4) request(host string, port int) ([]byte, error) {
	var buf bytes.Buffer

	buf.Write([]byte{socksVersion, socksConnect})
	_ = binary.Write(&buf, binary.BigEndian, uint16(port))

	ip, err := s.lookupAddr(host)
	if err != nil {
		return nil, err
	}

	_ = binary.Write(&buf, binary.BigEndian, ip)
	buf.WriteString(Ident)

	buf.WriteByte(0)

	return buf.Bytes(), nil
}

func (s socks4) requestSocks4a(host string, port int) []byte {
	var buf bytes.Buffer

	buf.Write([]byte{socksVersion, socksConnect})
	_ = binary.Write(&buf, binary.BigEndian, uint16(port))
	buf.Write([]byte{0, 0, 0, 1})
	buf.WriteString(Ident)
	buf.WriteString(host)

	buf.WriteByte(0)

	return buf.Bytes()
}

func (s socks4) Dial(network, addr string) (c net.Conn, err error) {
	if network != "tcp" && network != "tcp4" {
		return nil, ErrWrongConnType
	}

	c, err = s.dialer.Dial(network, s.url.Host)
	if err != nil {
		return nil, ErrDialFailed.Wrap(err)
	}
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return c, ErrWrongURL.WithArgs(addr).Wrap(err)
	}

	iport, err := strconv.Atoi(port)
	if err != nil {
		return c, ErrWrongURL.WithArgs(addr).Wrap(err)
	}

	var req []byte
	if s.url.Scheme == "socks4a" {
		req = s.requestSocks4a(host, iport)
	} else {
		req, err = s.request(host, iport)
		if err != nil {
			return c, err
		}
	}

	var i int
	i, err = c.Write(req)
	switch {
	case err != nil:
		return c, ErrSocksServer.Wrap(err)
	case i < minRequestLen:
		return c, ErrSocksServer.Wrap(io.ErrShortWrite)
	}

	var resp [8]byte
	i, err = c.Read(resp[:])
	switch {
	case err != nil && err != io.EOF:
		return c, ErrSocksServer.Wrap(err)
	case i != 8:
		return c, ErrSocksServer.Wrap(io.ErrUnexpectedEOF)
	}

	switch resp[1] {
	case accessGranted:
		return c, nil
	case accessIdentRequired, accessIdentFailed:
		return c, ErrIdentRequired.WithArgs(resp[1])
	case accessRejected:
		return c, ErrConnRejected
	default:
		return c, ErrUnknown.WithArgs(resp[1])
	}
}
