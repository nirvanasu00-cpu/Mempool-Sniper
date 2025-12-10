package main

import "fmt"

func main() {
    ch := make(chan int, 3) // 创建缓冲大小为3的通道
    ch <- 1
    ch <- 2
    ch <- 3
    close(ch) // 关闭通道，但缓冲区有 1, 2, 3 三个数据

    // 第一次读
    v1, ok1 := <-ch
    fmt.Printf("第一次读: value = %v, ok = %v\n", v1, ok1)

    // 第二次读
    v2, ok2 := <-ch
    fmt.Printf("第二次读: value = %v, ok = %v\n", v2, ok2)

    // 第三次读
    v3, ok3 := <-ch
    fmt.Printf("第三次读: value = %v, ok = %v\n", v3, ok3)

    // 第四次读 (此时缓冲区已空)
    v4, ok4 := <-ch
    fmt.Printf("第四次读: value = %v, ok = %v\n", v4, ok4)
}