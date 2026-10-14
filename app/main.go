package main

import (
	"fmt"
	"net"
	"os"
)

const (
	READ_BUF_SIZE = 1024
)

func handleConnection(conn net.Conn) {
	for {
		buf := make([]byte, READ_BUF_SIZE)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading to buffer: ", err.Error())
			return
		}
		conn.Write([]byte("+PONG\r\n"))
	}
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}

}
