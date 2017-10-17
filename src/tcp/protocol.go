package tcp

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
)

func dePacket(conn net.Conn, taskChan *chan []byte, buf []byte) ([]byte, error) {
	state := 0x00
	length := uint16(0)
	bufLength := int(0)
	var recvBuffer []byte
	cursor := uint16(0)
	tmpBuffer := make([]byte, 2)

	//	for {
	buffer, err := ioutil.ReadAll(conn)
	if err != nil {
		if err == io.EOF {
			fmt.Printf("closed: tcp client %s\n", conn.RemoteAddr().String())
			close(*taskChan)
			return nil, err
		}
	}
	//		n, err := conn.Read(buffer[lenc:])
	//		if n > 0 {
	//			lenc += n
	//		}
	//		if err != nil {
	//			if err != io.EOF {
	//				continue
	//			}
	//			fmt.Printf("closed: tcp client %s\n", conn.RemoteAddr().String())
	//			break
	//		}
	//	}
	//	if err != nil {
	//		fmt.Println(conn.RemoteAddr().String(), " connection error: ", err)
	//		return nil, err
	//	}
	lenc := len(buffer)
	bufLength = lenc
	if buf != nil && len(buf) != 0 {
		bufLength = len(buf) + bufLength
		buf = append(buf, buffer[:lenc]...)
	} else {
		buf = buffer
	}

	var i int
	for i = 0; i < bufLength; i = i + 1 {
		recvByte := buf[i]
		//		if err != nil {
		//			if err == io.EOF {
		//				fmt.Printf("closed: tcp client %s\n", conn.RemoteAddr().String())
		//				close(*taskChan)
		//				break
		//			} else {
		//				continue
		//			}
		//
		//		}
		switch state {
		case 0x00:
			tmpBuffer[0] = recvByte
			length += uint16(recvByte) * 256
			state++
		case 0x01:
			tmpBuffer[1] = recvByte
			length += uint16(recvByte)
			recvBuffer = make([]byte, length)
			state++
		case 0x02:
			tmpBuffer = append(tmpBuffer, recvByte)
			//			tmpBuffer[tmpCursor] = recvByte
			recvBuffer[cursor] = recvByte
			cursor++
			if cursor == length {
				*taskChan <- recvBuffer
				state = 0x00
				length = uint16(0)
				cursor = uint16(0)
				recvBuffer = nil
				tmpBuffer = make([]byte, 2)
			}
		}
	}
	if cursor != length {
		return tmpBuffer, nil
	}
	return nil, nil
}

func dePacketClient(conn net.Conn, taskChan *chan []byte) {
	state := 0x00
	length := uint16(0)
	var recvBuffer []byte
	cursor := uint16(0)
	tmpBuffer := make([]byte, 1024)
	reader := bufio.NewReader(conn)
	for {
		recvByte, err := reader.ReadByte()
		//		fmt.Println("receive :", recvByte, "status:", state)
		if err != nil {
			fmt.Println(err)
			if err == io.EOF {
				fmt.Printf("closed: tcp client %s\n", conn.RemoteAddr().String())
				close(*taskChan)
				return
			} else {
				continue
			}

		}
		switch state {
		case 0x00:
			tmpBuffer[0] = recvByte
			length += uint16(recvByte) * 256
			state++
		case 0x01:
			tmpBuffer[1] = recvByte
			length += uint16(recvByte)
			recvBuffer = make([]byte, length)
			state++
		case 0x02:
			tmpBuffer = append(tmpBuffer, recvByte)
			recvBuffer[cursor] = recvByte
			cursor++
			if cursor == length {
				*taskChan <- recvBuffer
				state = 0x00
				length = uint16(0)
				cursor = uint16(0)
				recvBuffer = nil
				tmpBuffer = make([]byte, 2)
			}
		}
	}
	if cursor != length {
		//		return tmpBuffer, nil
		println("cursor != length------------------------", tmpBuffer)
	}
}

func enPacket(packetType byte, sendBytes []byte) []byte {
	tempSlice := make([]byte, len(sendBytes)+1)
	tempSlice[0] = packetType
	copy(tempSlice[1:], sendBytes)
	packetLength := len(tempSlice) + 2
	result := make([]byte, packetLength)
	result[0] = byte(uint16(len(tempSlice)) >> 8)
	result[1] = byte(uint16(len(tempSlice)) & 0xFF)
	copy(result[2:], tempSlice)
	return result
}
