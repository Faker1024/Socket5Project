package main

import (
	"fmt"
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
	remoteAddr := client.RemoteAddr().String()
	fmt.Printf("Connection from %s\n", remoteAddr)
	_, err := client.Write([]byte("Hello world!\n"))
	if err != nil {
		fmt.Printf("write failed: %v", err)
		return
	}
	client.Close()
}
