package parser

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type Kind int

const (
	KindSimpleString Kind = iota
	KindError
	KindInteger
	KindBulkString
	KindArray
	KindNull
)

type Value struct {
	Kind  Kind
	Str   string
	Int   int64
	Array []Value
}

func readLine(reader *bufio.Reader) (string, error) {
	data, err := reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	if len(data) < 2 || data[len(data)-2] != '\r' {
		return "", errors.New("protocol error")
	}
	return string(data[:len(data)-2]), nil
}

func Parser(reader *bufio.Reader) (Value, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return Value{}, err
	}
	switch prefix {
	case '+':
		s, err := readLine(reader)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KindSimpleString, Str: s}, nil
	case '-':
		s, err := readLine(reader)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KindError, Str: s}, nil
	case '$':
		return parseBulk(reader)
	case '*':
		return parseArray(reader)
	default:
		return Value{}, errors.New("unknown type")
	}
}

func parseBulk(reader *bufio.Reader) (Value, error) {
	s, err := readLine(reader)
	if err != nil {
		return Value{}, err

	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return Value{}, errors.New("protocol error")
	}
	if n == -1 {
		return Value{Kind: KindNull}, nil
	}
	if n < -1 {
		return Value{}, errors.New("protocol error")
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return Value{}, err
	}
	// read \r\n
	crlf := make([]byte, 2)
	_, err = io.ReadFull(reader, crlf)
	if err != nil {
		return Value{}, err
	}
	if crlf[0] != '\r' || crlf[1] != '\n' {
		return Value{}, errors.New("protocol error")
	}
	return Value{Kind: KindBulkString, Str: string(buf)}, nil
}

func parseArray(reader *bufio.Reader) (Value, error) {
	s, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return Value{}, errors.New("protocol error")
	}
	if n == -1 {
		return Value{Kind: KindNull}, nil
	}
	if n < -1 {
		return Value{}, errors.New("protocol error")
	}
	result := make([]Value, n)
	for i := range n {
		v, err := Parser(reader)
		if err != nil {
			return Value{}, err
		}
		result[i] = v
	}
	return Value{Kind: KindArray, Array: result}, nil
}
