package tcp

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"
)

type Client struct {
	RAddr *net.TCPAddr
	CP    *ConnPool
	Conn  *net.TCPConn
}

var enPacketedData []byte
var enPacketChan = make(chan bool)

func NewClient(v interface{}) (Client, error) {
	vTyp = reflect.TypeOf(v)
	vVal = reflect.ValueOf(v)
	cli := Client{}
	err := cli.checkField()
	if err != nil {
		return cli, err
	}
	err = cli.checkMethod()
	if err != nil {
		return cli, err
	}
	cp, _ := cli.newConnPool(10, 2)
	cli.CP = cp
	return cli, nil
}

//连接池
type ConnPool struct {
	//互斥锁，保证资源安全
	mu sync.Mutex
	//通道，保存所有连接资源
	conns chan *net.TCPConn
	//判断池是否关闭
	closed bool
}

//连接池应该加入超时机制 timeout
//最大cap  初始化initNum
func (cli *Client) newConnPool(cap int, initNum int) (*ConnPool, error) {
	if cap <= 0 {
		return nil, errors.New("cap不能小于0")
	}
	cp := &ConnPool{
		mu:     sync.Mutex{},
		conns:  make(chan *net.TCPConn, cap),
		closed: false,
	}
	for i := 0; i < initNum; i++ {
		conn, err := net.DialTCP("tcp", nil, cli.RAddr)
		if err != nil && conn != nil {
			conn.Close()
			log.Println("net dial error :", err)
			//			return nil, errors.New("create conn 出错")
		}
		//将连接资源插入通道中
		cp.conns <- conn
		fmt.Println("create Connect ： ", i)
	}

	return cp, nil
}

//获取连接资源
func (cli *Client) getConn() (*net.TCPConn, error) {
	if cli.CP.closed {
		return nil, errors.New("连接池已关闭")
	}

	for {
		select {
		//从通道中获取连接资源
		case conn, ok := <-cli.CP.conns:
			if !ok {
				return nil, errors.New("连接池已关闭")
			}
			return conn, nil
		default:
			//如果无法从通道中获取资源，则重新创建一个资源返回
			conn, err := net.DialTCP("tcp", nil, cli.RAddr)
			if err != nil && conn != nil {
				conn.Close()
				return nil, errors.New("new conn 出错")
			}
			//将连接资源插入通道中
			cli.CP.conns <- conn
			return conn, nil
		}
	}
}

//连接资源放回池中
func (cli *Client) Put(conn *net.TCPConn) error {
	if cli.CP.closed {
		return errors.New("连接池已关闭")
	}

	select {
	//向通道中加入连接资源
	case cli.CP.conns <- conn:
		{
			return nil
		}
	default:
		{
			//如果无法加入，则关闭连接
			conn.Close()
			return errors.New("连接池已满")
		}
	}
}

func (cli *Client) checkField() error {
	Addr, err := commonFieldCheck()
	if err != nil {
		return err
	}
	cli.RAddr = Addr
	return nil
}

func (cli *Client) checkMethod() error {
	err := commonMethodCheck()
	if err != nil {
		return err
	}
	for i := 0; i < moduleNum; i++ {
		tempMethod := vTyp.Method(i)
		if moduleNames[tempMethod.Name[1:]] == nil {
			err := errors.New("init error: methods are defined incorrectly.")
			return err
		}
		if tempMethod.Type.In(1) != byteSliceType {
			err := errors.New("init error: method `" + tempMethod.Name + "` has wrong type of params.")
			return err
		}
		mapNameFunc[tempMethod.Name[1:]] = tempMethod.Func
	}
	return nil
}

//func (cli *Client) dial() error {
//	if cli.Conn == nil {
//		conn, err := net.DialTCP("tcp", nil, cli.RAddr)
//		if err != nil {
//			return err
//		}
//		cli.Conn = conn
//	}
//	return nil
//}

func (cli *Client) Send(typ byte, data []byte) error {
	go cli.enPacket(typ, data)
	//	err := cli.dial()
	conn, err := cli.getConn()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	<-enPacketChan
	//	if err != nil {
	//		return err
	//	}
	cli.send(conn)
	taskChan := make(chan []byte, 1)
	go cli.Process(taskChan)
	go dePacketClient(conn, &taskChan)
	err = cli.Put(conn)
	if err != nil {
		fmt.Println(err.Error())
	}
	return nil
}

func (cli *Client) Process(taskChan chan []byte) {
	for recvBuffer := range taskChan {
		packetType := recvBuffer[0]
		data := recvBuffer[1:]
		mapNameFunc[originMap[packetType]].Call([]reflect.Value{vVal, reflect.ValueOf(data)})
	}
}

func (cli *Client) enPacket(typ byte, data []byte) {
	enPacketedData = enPacket(typ, data)
	enPacketChan <- true
}

func (cli *Client) send(conn *net.TCPConn) {
	bufferWriter := bufio.NewWriter(conn)
	bufferWriter.Write(enPacketedData)
	bufferWriter.Flush()
}
