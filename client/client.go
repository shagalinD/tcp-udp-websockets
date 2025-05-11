package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	ClientUDP()
	go ClientTCP()

	select {}
}

func ClientUDP() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:9090")
	if err != nil {
			fmt.Println("Error resolving address:", err)
			return
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		fmt.Println("error resolving address: ", err.Error())
		return 
	}

	defer conn.Close()
	data := "Hello UDP server!!"

	if _, err := conn.Write([]byte(data)); err != nil {
		fmt.Println("error sending datagram: ", err)
		return 
	}

	buf := make([]byte, len(data))
	if _, err := conn.Read(buf); err != nil {
		fmt.Println("error reading datagram", err)
		return 
	}

	fmt.Println("received from server: ", string(buf))
}

func ClientTCP() {
	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		fmt.Println("error connecting", err.Error())
		os.Exit(1)
	}

	defer conn.Close() 
	fmt.Println("enter text to send: ")
	consoleScanner := bufio.NewScanner(os.Stdin)

	for consoleScanner.Scan() {
		text := consoleScanner.Text()
		conn.Write([]byte(text + "\n"))
		response, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println("error reading: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("Server says: " + response)
		fmt.Println("Enter more text to send: ")
	}

	if err := consoleScanner.Err(); err != nil {
		fmt.Println("Error reading from console: ", err.Error())
	}
}