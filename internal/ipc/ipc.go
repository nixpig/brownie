package ipc

import "net"

type closer func() error

func NewSender(sockAddr string) (chan []byte, closer, error) {
	ch := make(chan []byte, 1024)

	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			b := <-ch
			conn.Write(b)
		}
	}()

	return ch, conn.Close, nil
}

func NewReceiver(sockAddr string) (chan []byte, closer, error) {
	ch := make(chan []byte, 1024)

	listener, err := net.Listen("unix", sockAddr)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		b := make([]byte, 1024)

		for {
			n, err := conn.Read(b)
			if err != nil || n == 0 {
				continue
			}

			ch <- b[:n]
		}
	}()

	return ch, listener.Close, nil
}