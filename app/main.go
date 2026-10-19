package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/parser"
	"github.com/codecrafters-io/redis-starter-go/app/types"
	"github.com/codecrafters-io/redis-starter-go/app/utils"
)

// handles connection
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
		args, err := utils.Argv(value)
		if err != nil {
			conn.Write(utils.EncodeError("ERR protocol error"))
			continue
		}
		switch strings.ToUpper(args[0]) {
		case string(types.CommandPing):
			conn.Write(utils.EncodeSimple("PONG"))
		case string(types.CommandEcho):
			if len(args) < 2 {
				conn.Write(utils.EncodeError("ERR wrong number of arguments for 'echo' command"))
				continue
			}
			conn.Write(utils.EncodeBulk(args[1]))
		default:
			conn.Write(utils.EncodeError("ERR unknown command"))
		}
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
