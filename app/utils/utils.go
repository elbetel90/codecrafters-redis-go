package utils

import (
	"errors"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/parser"
)

func EncodeSimple(s string) []byte {
	return []byte("+" + s + "\r\n")
}

func EncodeBulk(s string) []byte {
	return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(s), s)
}

func EncodeError(s string) []byte {
	return []byte("-" + s + "\r\n")
}

func EncodeNullBulk() []byte {
	return []byte("$-1\r\n")
}

func Argv(value parser.Value) ([]string, error) {
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
