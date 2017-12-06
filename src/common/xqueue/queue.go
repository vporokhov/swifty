package xqueue

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"encoding/json"
)

const (
	qChunk	= 1024
)

type Queue struct {
	sk	*os.File
	pfd	int
}

func (q *Queue)GetId() string {
	return strconv.Itoa(q.pfd)
}

func (q *Queue)FDS() string {
	return fmt.Sprintf("%d:%d", q.pfd, q.sk.Fd())
}

func (q *Queue)Started() {
	syscall.Close(q.pfd)
	q.pfd = -1
}

func (q *Queue)Close() {
	if q.pfd != -1 {
		syscall.Close(q.pfd)
		q.pfd = -1
	}
	q.sk.Close()
}

func split(buf []byte, cs int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/cs+1)
	for len(buf) >= cs {
		chunk, buf = buf[:cs], buf[cs:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf)
	}
	return chunks
}

func (q *Queue)Send(msg interface{}) error {
	dat, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode message error: %s", err.Error())
	}

	return q.SendBytes(dat)
}

func (q *Queue)SendBytes(bts []byte) error {
	/*
	 * We'll send data in chunks. For the receiver to detect
	 * the last packet in the queue we make its size less than
	 * the chunk's one.
	 */
	if len(bts) % qChunk == 0 {
		bts = append(bts, byte(0))
	}

	chunks := split(bts, qChunk)
	for _, ch := range chunks {
		_, err := q.sk.Write(ch)
		if err != nil {
			return fmt.Errorf("send message error: %s", err.Error())
		}
	}

	return nil
}

func (q *Queue)recvBytes() ([]byte, error) {
	msg := make([]byte, 0, 4096)
	bts := make([]byte, qChunk)

	for {
		l, err := q.sk.Read(bts)
		if err != nil {
			return nil, fmt.Errorf("recv message error: %s", err.Error())
		}

		if l < qChunk {
			if bts[l-1] == byte(0) {
				l -= 1
			}
			msg = append(msg, bts[:l]...)
			break
		}

		msg = append(msg, bts...)
	}

	return msg, nil
}

func (q *Queue)RecvStr() (string, error) {
	bts, err := q.recvBytes()
	if err != nil {
		return "", err
	}

	return string(bts), nil
}

func (q *Queue)Recv(in interface{}) error {
	dat, err := q.recvBytes()
	if err != nil {
		return err
	}

	err = json.Unmarshal(dat, in)
	if err != nil {
		return fmt.Errorf("decode message error: %s", err.Error())
	}

	return nil
}

func MakeQueue() (*Queue, error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		return nil, fmt.Errorf("socketpair error: %s", err.Error())
	}

	syscall.CloseOnExec(fds[1])
	return &Queue{sk: os.NewFile(uintptr(fds[1]), "queue"), pfd: fds[0]}, nil
}

func OpenQueue(id string) (*Queue, error) {
	fd, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("bad qid: %s", err.Error())
	}

	return &Queue{sk: os.NewFile(uintptr(fd), "queue"), pfd: -1}, nil
}
