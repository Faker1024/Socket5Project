package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func main() {

	listen, err := net.Listen(`tcp`, `:1080`)
	if err != nil {
		fmt.Printf("Listen failed: %v\n", err)
		return
	}

	for {
		client, err := listen.Accept()
		if err != nil {
			fmt.Printf("Accept failed: %v", err)
			continue
		}
		go process(client)
	}

}

func process(client net.Conn) {
	if err := Socks5Auth(client); err != nil {
		fmt.Println("auth error: ", err)
		client.Close()
		return
	}
	target, err := Socks5Connect(client)
	if err != nil {
		fmt.Println("connect error: ", err)
		client.Close()
		return
	}
	Socks5Forward(client, target)
}

func Socks5Forward(client net.Conn, target net.Conn) {
	forward := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		io.Copy(client, target)
	}
	go forward(client, target)
	go forward(target, client)
}

func Socks5Connect(client net.Conn) (net.Conn, error) {
	bytes := make([]byte, 256)
	n, err := io.ReadFull(client, bytes[:4])
	if n != 4 {
		return nil, errors.New("read Header : " + err.Error())
	}
	ver, cmd, _, atyp := int(bytes[0]), int(bytes[1]), int(bytes[2]), int(bytes[4])
	if ver != 5 || cmd != 1 {
		return nil, errors.New("vail ver/cmd")
	}
	addr := ""
	switch atyp {
	case 1:
		n, err = io.ReadFull(client, bytes[:4])
		if n != 4 {
			return nil, errors.New("invalid IPv4: " + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", bytes[0], bytes[1], bytes[2], bytes[3])
	case 3:
		n, err = io.ReadFull(client, bytes[:4])
		if n != 1 {
			return nil, errors.New("invalid hostName: " + err.Error())
		}
		addrLen := int(bytes[0])
		n, err = io.ReadFull(client, bytes[:addrLen])
		if n != addrLen {
			return nil, errors.New("invalid hostName: " + err.Error())
		}
		addr = string(bytes[:addrLen])
	case 4:
		return nil, errors.New("IPv6: no supported yet")
	default:
		return nil, errors.New("invalid atyp")
	}
	n, err = io.ReadFull(client, bytes[:2])
	if n != 2 {
		return nil, errors.New("read port: " + err.Error())
	}
	port := binary.BigEndian.Uint16(bytes[:2])
	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	dest, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, errors.New("dial dst: " + err.Error())
	}
	n, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		dest.Close()
		return nil, errors.New("write rsp: " + err.Error())
	}
	return dest, nil
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)

	/*读取VER和NMETHODS*/
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		return errors.New("reading header: " + err.Error())
	}
	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invalid version")
	}

	/*读取METHODS列表*/
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		return errors.New("reading methods: " + err.Error())
	}

	/*无需认证*/
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write rsp: " + err.Error())
	}
	return nil
}
