package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/parser"
	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/codecrafters-io/redis-starter-go/app/types"
	"github.com/codecrafters-io/redis-starter-go/app/utils"
)

// handles connection
func handleConnection(conn net.Conn, store *store.Store) {
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
		case string(types.CommandGet):
			if len(args) < 2 {
				conn.Write(utils.EncodeError("ERR wrong number of arguments for 'get' command"))
				continue
			}
			v, ok := store.Get(args[1])
			if !ok {
				conn.Write(utils.EncodeNullBulk())
				continue
			}
			conn.Write(utils.EncodeBulk(v))
		case string(types.CommandSet):
			if len(args) < 3 {
				conn.Write(utils.EncodeError("ERR wrong number of arguments for 'set' command"))
				continue
			}
			var expiry time.Time
			if len(args) > 3 {
				if len(args) < 5 {
					conn.Write(utils.EncodeError("ERR syntax error"))
					continue
				}
				n, err := strconv.Atoi(args[4])
				if err != nil {
					conn.Write(utils.EncodeError("ERR value is not an integer or out of range"))
					continue
				}
				switch strings.ToUpper(args[3]) {
				case string(types.OptionalSetCommandArgEX):
					expiry = time.Now().Add(time.Duration(n) * time.Second)
				case string(types.OptionalSetCommandArgPX):
					expiry = time.Now().Add(time.Duration(n) * time.Millisecond)
				default:
					conn.Write(utils.EncodeError("ERR syntax error"))
					continue
				}
			}
			store.Set(args[1], args[2], expiry)
			conn.Write(utils.EncodeSimple("OK"))
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

	st := store.NewStore()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn, st)
	}

}
