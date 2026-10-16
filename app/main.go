package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/parser"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		value, err := parser.Parser(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error parsing: ", err.Error())
			}
			return
		}
		_ = value
		j, _ := json.MarshalIndent(value, " ", " ")
		fmt.Println(string(j))
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
