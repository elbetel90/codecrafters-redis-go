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

func processCommand(conn net.Conn, store *store.Store, args []string) {
	switch strings.ToUpper(args[0]) {
	case string(types.CommandPing):
		conn.Write(utils.EncodeSimple("PONG"))
	case string(types.CommandEcho):
		if len(args) < 2 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'echo' command"))
			return
		}
		conn.Write(utils.EncodeBulk(args[1]))
	case string(types.CommandGet):
		if len(args) < 2 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'get' command"))
			return
		}
		v, ok := store.Get(args[1])
		if !ok {
			conn.Write(utils.EncodeNullBulk())
			return
		}
		conn.Write(utils.EncodeBulk(v))
	case string(types.CommandSet):
		if len(args) < 3 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'set' command"))
			return
		}
		var expiry time.Time
		if len(args) > 3 {
			if len(args) < 5 {
				conn.Write(utils.EncodeError("ERR syntax error"))
				return
			}
			n, err := strconv.Atoi(args[4])
			if err != nil {
				conn.Write(utils.EncodeError("ERR value is not an integer or out of range"))
				return
			}
			switch strings.ToUpper(args[3]) {
			case string(types.OptionalSetCommandArgEX):
				expiry = time.Now().Add(time.Duration(n) * time.Second)
			case string(types.OptionalSetCommandArgPX):
				expiry = time.Now().Add(time.Duration(n) * time.Millisecond)
			default:
				conn.Write(utils.EncodeError("ERR syntax error"))
				return
			}
		}
		store.Set(args[1], args[2], expiry)
		conn.Write(utils.EncodeSimple("OK"))
	case string(types.CommandRPush):
		if len(args) < 3 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'rpush' command"))
			return
		}
		n, err := store.RPush(args[1], args[2:]...)
		if err != nil {
			conn.Write(utils.EncodeError("ERR " + err.Error()))
			return
		}
		conn.Write(utils.EncodeInt(n))
	case string(types.CommandLRange):
		if len(args) < 4 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'lrange' command"))
			return
		}
		startIndex, err := strconv.Atoi(args[2])
		if err != nil {
			conn.Write(utils.EncodeError("ERR wrong start index type"))
			return
		}
		stopIndex, err := strconv.Atoi(args[3])
		if err != nil {
			conn.Write(utils.EncodeError("ERR wrong start index type"))
			return
		}
		v := store.LRange(args[1], startIndex, stopIndex)
		if len(v) == 0 {
			conn.Write(utils.EncodeEmptyArray())
			return
		}
		conn.Write(utils.EncodeArray(v))
	case string(types.CommandLPush):
		if len(args) < 3 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'lpush' command"))
			return
		}
		n, err := store.LPush(args[1], args[2:]...)
		if err != nil {
			conn.Write(utils.EncodeError("ERR " + err.Error()))
			return
		}
		conn.Write(utils.EncodeInt(n))
	case string(types.CommandLLen):
		if len(args) < 2 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'llen' command"))
			return
		}
		n := store.LLen(args[1])
		conn.Write(utils.EncodeInt(n))
	case string(types.CommandLPop):
		if len(args) < 2 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'lpop' command"))
			return
		}
		count := 1
		withCount := false
		if len(args) >= 3 {
			n, err := strconv.Atoi(args[2])
			if err != nil || n < 0 {
				conn.Write(utils.EncodeError("ERR value is not an integer or out of range"))
				return
			}
			count = n
			withCount = true
		}
		item, exists := store.LPop(args[1], count)
		if !exists {
			conn.Write(utils.EncodeNullBulk())
			return
		}
		if withCount {
			conn.Write(utils.EncodeArray(item))
			return
		}
		conn.Write(utils.EncodeBulk(item[0]))
	case string(types.CommandBLpop):
		if len(args) < 3 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'blpop' command"))
			return
		}
		secs, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			conn.Write(utils.EncodeError("ERR timeout is not a float or out of range"))
			return
		}
		val, ok := store.BLpop(args[1], time.Duration(secs*float64(time.Second)))
		if !ok {
			conn.Write(utils.EncodeNullArray())
			return
		}
		conn.Write(utils.EncodeArray([]string{args[1], val}))
	case string(types.CommandType):
		if len(args) < 2 {
			conn.Write(utils.EncodeError("ERR wrong number of arguments for 'type' command"))
			return
		}
		conn.Write(utils.EncodeSimple(store.Type(args[1])))
	default:
		conn.Write(utils.EncodeError("ERR unknown command"))
	}
}

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
		processCommand(conn, store, args)
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
