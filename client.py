package main
import (
    "fmt"
    "net"
    "bufio"
    "time"
    "sync"
    "runtime"
)

type worker struct {
    Func func()
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    var wg sync.WaitGroup

    channels := make(chan worker, 1000)

    for i := 0; i < 190; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for ch := range channels {
                //reflect.ValueOf(ch.Func).Call(ch.Args)
                ch.Func()
            }
        }()
    }

    for i := 0; i < 200; i++ {
        wk := worker{
            Func: func() {
                conn, err := net.Dial("tcp", ":8088")
                if err != nil {
                    panic(err)
                }
                for {
                    fmt.Fprintf(conn, "hello server\n")
                    data , err := bufio.NewReader(conn).ReadString('\n')
                    if err != nil {
                        panic(err)
                    }
                    //count++
                    //fmt.Println(count)
                    _ = data
                    time.Sleep(10*time.Millisecond)
                }
            },
        }
        channels <- wk
    }
    close(channels)
    wg.Wait()
}
