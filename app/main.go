package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/parser"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

func encodeSimple(s string) []byte {
	return []byte("+" + s + "\r\n")
}

func encodeBulk(s string) []byte {
	return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(s), s)
}

func encodeError(s string) []byte {
	return []byte("-" + s + "\r\n")
}

func argv(value parser.Value) ([]string, error) {
	if value.Kind != parser.KindArray || len(value.Array) == 0 {
		return nil, errors.New("expected command array")
	}
	args := make([]string, len(value.Array))
	for i, v := range value.Array {
		if v.Kind != parser.KindBulkString {
			return nil, errors.New("expected bulk string argument")
		}
		args[i] = v.Str
	}
	return args, nil
}

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
		args, err := argv(value)
		if err != nil {
			conn.Write(encodeError("ERR protocol error"))
			continue
		}
		switch strings.ToUpper(args[0]) {
		case string(types.CommandPing):
			conn.Write(encodeSimple("PONG"))
		case string(types.CommandEcho):
			if len(args) < 2 {
				conn.Write(encodeError("ERR wrong number of arguments for 'echo' command"))
				continue
			}
			conn.Write(encodeBulk(args[1]))
		default:
			conn.Write(encodeError("ERR unknown command"))
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
