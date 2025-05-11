package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const PORT_TCP = ":8080"
const PORT_UDP = ":9090"

func main() {
	go ListenTCP()
	go ListenUDP()

	select {}
}

func ListenUDP() {
	conn, err := net.ListenPacket("udp", ":9090")

	if err != nil {
		fmt.Println("error on listening packets: ", err.Error())
		os.Exit(1)
	}

	defer conn.Close()

	fmt.Println("Listening packets on :9090")

	buf := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFrom(buf)

		if err != nil {
			fmt.Println("error on reading from packets: ", err.Error())
			continue
		}

		if _, err := conn.WriteTo(buf[:n], addr); err != nil {
			fmt.Println("error on writing datagram: ", err.Error())
			continue
		}
	}
}

func ListenTCP() {
	listener, err := net.Listen("tcp", PORT_TCP)

	if err != nil {
		fmt.Println("error on listening: ", err.Error())
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("Server is listening on port " + PORT_TCP)

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println("error on accepting: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("Connected with ", conn.RemoteAddr().String())

		go HandleRequest(conn)
	}
}

func HandleRequest(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn) 

	for scanner.Scan() {
		clientMessage := scanner.Text() 
		fmt.Printf("Received from client: %s\n", clientMessage)

		conn.Write([]byte("Message received\n"))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error reading: ", err.Error())
	}
}