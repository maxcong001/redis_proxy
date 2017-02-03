package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/garyburd/redigo/redis"
)

//glob var
var (
	pool          *redis.Pool
	redisServer   = flag.String("redisServer", ":6379", "")
	redisPassword = flag.String("redisPassword", "", "")
        count uint64 = 1
        last_count uint64 = 1
)

//init a pool
func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		MaxActive:   300,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			/*
			   if _, err := c.Do("AUTH", password); err != nil {
			       c.Close()
			       return nil, err
			   }
			*/
			return c, err
		},

		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

const (
	MAX_CONN_NUM = 99
)

//redis Goroutine
func redis_routine(conn_socket net.Conn) {
	defer conn_socket.Close()
	buf := make([]byte, 1024)
	for {
		_, err := conn_socket.Read(buf)
		if err != nil {
			//println("Error reading:", err.Error())
			return
		}

		conn := pool.Get()
		defer conn.Close()
		//redis command
		v, err := conn.Do("SET", "pool", "test")
		if err != nil {
			fmt.Println(err)
			return
		}
		//fmt.Println(v)
		v, err = redis.String(conn.Do("GET", "pool"))
		if err != nil {
			fmt.Println(err)
			return
		}
                conn.Close()
                count++
		//fmt.Println(v)
		_ = v
		//send reply
		_, err = conn_socket.Write(buf)
		if err != nil {
			//println("Error send reply:", err.Error())
			return
		}
	}
}

//initial listener and run
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	listener, err := net.Listen("tcp", "0.0.0.0:8088")
	if err != nil {
		fmt.Println("error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("running ...\n")

	flag.Parse()
	pool = newPool(*redisServer, *redisPassword)

	var cur_conn_num int = 0
	conn_chan := make(chan net.Conn)
	ch_conn_change := make(chan int)

	go func() {
		for conn_change := range ch_conn_change {
			cur_conn_num += conn_change
		}
	}()
	go func() {
		for _ = range time.Tick(100*time.Millisecond) {
			fmt.Printf("cur conn num: %d, package %d, %d package per sec\r", cur_conn_num, count, (count-last_count)*10)
                        last_count = count
		}
	}()
	for i := 0; i < MAX_CONN_NUM; i++ {
		go func() {
			for conn := range conn_chan {
				ch_conn_change <- 1
				redis_routine(conn)
				ch_conn_change <- -1
			}
		}()
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			println("Error accept:", err.Error())
			return
		}
		conn_chan <- conn
	}
}
