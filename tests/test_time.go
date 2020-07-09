package main

import (
	"fmt"
	_ "log"
	"time"
)

func main() {
	fmt.Printf("unix %v, %T\n", time.Now().Unix(), time.Now().Unix())
	fmt.Printf("utc %v, %T\n", time.Now().UTC(), time.Now().UTC())
	t := time.Unix(1496887096, 0)
	fmt.Printf("%v, %T\n", t, t)
	fmt.Println(t.Format(time.UnixDate))
	fmt.Println(t.Format(time.ANSIC))
	fmt.Println(t.Format(time.RFC3339))
}
