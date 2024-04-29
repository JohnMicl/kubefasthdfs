package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"kubefasthdfs/config"
	"kubefasthdfs/logger"
	"kubefasthdfs/rocksdb"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

func main() {
	// nums1 := []int{1, 2, 100, 2, 3, 4, 7, 6, 5, 4, 1002, 10003, 30001, 10, 5, 2}
	// mymergesort1(nums1)
	// fmt.Println(nums1)
	// // nums2 := []int{1, 2, 100, 2, 3, 4, 7, 6, 5, 4, 1002, 10003, 30001, 10, 5, 2}
	// // mymergesort2(nums2)
	// nums3 := []int{1, 2, 100, 2, 3, 4, 7, 6, 5, 4, 1002, 10003, 30001, 10, 5, 2}
	// sort.Ints(nums3)
	// fmt.Println(nums3)
	err := logger.InitLogger(config.Config.LogInfo)
	if err != nil {
		panic(fmt.Sprintf("logger init err:%v", err))
	}
	logger.Logger.Info("logger init success!")
	test := "raft_log_"
	rbd, err := rocksdb.NewRocksDBStore("/tmp/testrocksdb", rocksdb.LRUCacheSize, rocksdb.WriteBufferSize)
	if err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
	num := int64(12322)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(num))
	test += string(buf)
	rbd.Put(test, []byte(test), false)

	data, err := rbd.Get(test)
	if err != nil {
		fmt.Println("Error2")
		os.Exit(1)
	}
	fmt.Println(data.([]byte), len(data.([]byte)), len("raft_log_"), (data.([]byte))[len("raft_log_"):])
	index := (data.([]byte))[len("raft_log_"):]
	if len(index) == 8 {
		num = int64(binary.BigEndian.Uint64(index))
		fmt.Println(num)
	}
}

func mymergesort1(nums []int) {
	dst := make([]int, len(nums))
	for seq := 1; seq < len(nums); seq++ {
		for start := 0; start < len(nums); start += seq * 2 {
			mid := min(start+seq, len(nums))
			end := min(start+2*seq, len(nums))
			j := mid
			i := start
			k := start
			for i < mid || j < end {
				if j < end && nums[j] < nums[i] {
					dst[k] = nums[j]
					k++
					j++
				} else {
					dst[k] = nums[i]
					k++
					i++
				}
			}
		}
		tmp := nums
		nums = dst
		dst = tmp
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pulseTest2() {
	dowork := func(done <-chan interface{}) (<-chan interface{}, <-chan int) {
		heartbeatStream := make(chan interface{})
		workStream := make(chan int)

		go func() {
			defer close(heartbeatStream)
			defer close(workStream)

			for i := 0; i < 10; i++ {
				select {
				case heartbeatStream <- struct{}{}:
					//default:
				}
				select {
				case <-done:
					return
				case workStream <- rand.Intn(10):
				}
			}
		}()
		return heartbeatStream, workStream
	}

	done := make(chan interface{})
	defer close(done)

	heartbeat, results := dowork(done)
	for {
		select {
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			fmt.Println("pulse")
		case r, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("results %v\n", r)
		}
	}
}

func pulseTest1() {
	dowork := func(done <-chan interface{}, pulseInterval time.Duration) (<-chan interface{}, <-chan time.Time) {
		heartbeat := make(chan interface{})
		results := make(chan time.Time)

		go func() {
			// defer close(heartbeat)
			// defer close(results)

			pulse := time.Tick(pulseInterval)
			workGen := time.Tick(2 * pulseInterval)

			sendPulse := func() {
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			}

			sendResult := func(r time.Time) {
				for {
					select {
					// case <-done:
					// 	return
					case <-pulse:
						sendPulse()
					case results <- r:
						return
					}
				}
			}

			for i := 0; i < 2; i++ {
				select {
				case <-done:
					return
				case <-pulse:
					sendPulse()
				case r := <-workGen:
					sendResult(r)
				}
			}
		}()
		return heartbeat, results
	}

	done := make(chan interface{})
	time.AfterFunc(10*time.Second, func() { close(done) })

	const timeout = 2 * time.Second
	heartbeat, results := dowork(done, timeout/2)
	for {
		select {
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			fmt.Println("pulse")
		case r, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("results %v\n", r.Second())
		case <-time.After(timeout):
			fmt.Println("work not health")
			return
		}
	}
}

func pulseTest() {
	dowork := func(done <-chan interface{}, pulseInterval time.Duration) (<-chan interface{}, <-chan time.Time) {
		heartbeat := make(chan interface{})
		results := make(chan time.Time)

		go func() {
			defer close(heartbeat)
			defer close(results)

			pulse := time.Tick(pulseInterval)
			workGen := time.Tick(3 * pulseInterval)

			sendPulse := func() {
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			}

			sendResult := func(r time.Time) {
				for {
					select {
					case <-done:
						return
					case <-pulse:
						sendPulse()
					case results <- r:
						return
					}
				}
			}

			for {
				select {
				case <-done:
					return
				case <-pulse:
					sendPulse()
				case r := <-workGen:
					sendResult(r)
				}
			}
		}()
		return heartbeat, results
	}

	done := make(chan interface{})
	time.AfterFunc(30*time.Second, func() { close(done) })

	const timeout = 2 * time.Second
	heartbeat, results := dowork(done, timeout/2)
	for {
		select {
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			fmt.Println("pulse")
		case r, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("results %v\n", r.Second())
		case <-time.After(timeout):
			return
		}
	}
}

func testcontext() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context) {
		now := time.Now()
		fmt.Println("child running!")
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(15*time.Second))
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("CHILD PASS%+v\n", time.Since(now))
				fmt.Println("child done!")
				return
			}
		}
	}(ctx)

	time.Sleep(10 * time.Second)
	cancel()

	fmt.Println("main abort child done!")
	time.Sleep(1 * time.Second)
}

func printGreeting(done <-chan interface{}) error {
	greeting, err := genGreenting(done)
	if err != nil {
		return err
	}
	fmt.Printf("%s word!\n", greeting)
	return nil
}

func printFarewell(done <-chan interface{}) error {
	farewell, err := genFarewell(done)
	if err != nil {
		return err
	}
	fmt.Printf("%s word!\n", farewell)
	return nil
}

func genFarewell(done <-chan interface{}) (string, error) {
	switch locale, err := locale(done); {
	case err != nil:
		return "", err
	case locale == "EN/US":
		return "Goodbye", nil
	}
	return "", fmt.Errorf("UNSUPPORTED LOCALE")
}

func genGreenting(done <-chan interface{}) (string, error) {
	switch locale, err := locale(done); {
	case err != nil:
		return "", err
	case locale == "EN/US":
		return "Hello", nil
	}
	return "", fmt.Errorf("UNSUPPORTED LOCALE")
}

func locale(done <-chan interface{}) (string, error) {
	select {
	case <-done:
		return "", fmt.Errorf("cancled")
	case <-time.After(1 * time.Second):
	}
	return "EN/US", nil
}

func bridgeChan() {

	orDone := func(done <-chan interface{}, c <-chan int) <-chan int {
		valStream := make(chan int)
		go func() {
			defer close(valStream)
			for {
				select {
				case <-done:
					return
				case v, ok := <-c:
					if !ok {
						return
					}
					select {
					case valStream <- v:
					case <-done:
						return
					}
				}
			}
		}()
		return valStream
	}

	bridge := func(done <-chan interface{}, chanstream <-chan <-chan int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for {
				var s <-chan int
				select {
				case ms, ok := <-chanstream:
					if !ok {
						return
					}
					s = ms
				case <-done:
					return
				}

				for val := range orDone(done, s) {
					select {
					case stream <- val:
					case <-done:
					}
				}
			}
		}()
		return stream
	}

	genvals := func() <-chan <-chan int {
		chanstream := make(chan (<-chan int))
		go func() {
			defer close(chanstream)
			for i := 0; i < 10; i++ {
				s := make(chan int, 1)
				s <- i
				close(s)
				chanstream <- s
			}
		}()
		return chanstream
	}

	for v := range bridge(nil, genvals()) {
		fmt.Printf("v=%d\n", v)
	}
}

func pipelinetee() {

	orDone := func(done <-chan interface{}, c <-chan int) <-chan int {
		valStream := make(chan int)
		go func() {
			defer close(valStream)
			for {
				select {
				case <-done:
					return
				case v, ok := <-c:
					if !ok {
						return
					}
					select {
					case valStream <- v:
					case <-done:
						return
					}
				}
			}
		}()
		return valStream
	}

	tee := func(done <-chan interface{}, in <-chan int) (<-chan int, <-chan int) {
		out1 := make(chan int)
		out2 := make(chan int)

		go func() {
			defer close(out1)
			defer close(out2)
			for val := range orDone(done, in) {
				var out1, out2 = out1, out2
				for i := 0; i < 2; i++ {
					select {
					case <-done:
					case out1 <- val:
						out1 = nil
					case out2 <- val:
						out2 = nil
					}
				}
			}
		}()
		return out1, out2
	}

	take := func(done <-chan interface{}, input <-chan int, num int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					fmt.Println("Take Break Down By main")
					return
				case v := <-input:
					stream <- v
				}
			}
		}()
		return stream
	}

	repeator := func(done <-chan interface{}, inte ...int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for {
				for _, v := range inte {
					select {
					case <-done:
						fmt.Println("Generator Break Down By main")
						return
					case stream <- v:
					}
				}
			}
		}()
		return stream
	}

	done := make(chan interface{})
	defer close(done)

	out1, out2 := tee(done, take(done, repeator(done, 1, 2), 4))

	for val1 := range out1 {
		fmt.Printf("out1 = %v, out2 = %v\n", val1, <-out2)
	}
}

func pipelineRepeatFn() {
	repeatorFn := func(done <-chan interface{}, fn func() int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for {
				select {
				case <-done:
					fmt.Println("Generator Break Down By main")
					return
				case stream <- fn():
				}
			}
		}()
		return stream
	}

	take := func(done <-chan interface{}, input <-chan int, num int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					fmt.Println("Take Break Down By main")
					return
				case v := <-input:
					stream <- v
				}
			}
		}()
		return stream
	}

	rand := func() int { return rand.Int() }

	done := make(chan interface{})
	for v := range take(done, repeatorFn(done, rand), 10) {
		fmt.Printf("%v\n", v)
	}
	close(done)
}

func pipelinewithgenerate() {

	repeator := func(done <-chan interface{}, inte ...int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for {
				for _, v := range inte {
					select {
					case <-done:
						fmt.Println("Generator Break Down By main")
						return
					case stream <- v:
					}
				}
			}
		}()
		return stream
	}

	take := func(done <-chan interface{}, input <-chan int, num int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					fmt.Println("Take Break Down By main")
					return
				case v := <-input:
					stream <- v
				}
			}
		}()
		return stream
	}

	done := make(chan interface{})
	defer close(done)
	for v := range take(done, repeator(done, 1), 10) {
		fmt.Printf("%v\n", v)
	}
}

func pipelinewithchannel() {
	done := make(chan interface{})
	generator := func(done <-chan interface{}, inte ...int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for _, v := range inte {
				select {
				case <-done:
					fmt.Println("Generator Break Down By main")
					return
				case stream <- v:
				}
			}
		}()
		return stream
	}

	mutiply := func(done <-chan interface{}, input <-chan int, m int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for v := range input {
				select {
				case <-done:
					fmt.Println("Mutiply Break Down By main")
					return
				case stream <- v * m:
				}
			}
		}()
		return stream
	}

	add := func(done <-chan interface{}, input <-chan int, a int) <-chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for v := range input {
				select {
				case <-done:
					fmt.Println("Add Break Down By main")
				case stream <- v + a:
				}
			}
		}()
		return stream
	}

	for v := range mutiply(done, add(done, generator(done, 1, 2, 3, 4, 5), 1), 2) {
		fmt.Println(v)
		if v == 6 {
			close(done)
		}
	}
	time.Sleep(1 * time.Second)
	fmt.Println("Waiting")

}

func pipelinetest() {
	mutiply := func(muti int, nums []int) []int {
		result := make([]int, len(nums))
		for i, v := range nums {
			result[i] = v * muti
		}
		return result
	}

	add := func(a int, nums []int) []int {
		result := make([]int, len(nums))
		for i, v := range nums {
			result[i] = v + a
		}
		return result
	}
	ints := []int{1, 2, 3}
	result := add(2, mutiply(2, add(1, ints)))
	fmt.Println(result)
}

type Result struct {
	Error    error
	Response *http.Response
}

func checkError() {
	checkStatus := func(done <-chan interface{}, urls ...string) <-chan Result {
		responses := make(chan Result)
		go func() {
			defer close(responses)
			for _, url := range urls {
				var result Result
				resp, err := http.Get(url)
				result = Result{Error: err, Response: resp}
				select {
				case <-done:
					//fmt.Println("Done by main")
					return
				case responses <- result:
				}
			}
		}()
		return responses
	}

	done := make(chan interface{})
	defer close(done)

	urls := []string{"A", "https://www.baidu.com", "https://badhostsdasd", "V", "E", "F"}
	responses := checkStatus(done, urls...)
	errcnt := 0
	for resp := range responses {
		if resp.Error != nil {
			fmt.Printf("error: %v\n", resp.Error)
			errcnt++
			if errcnt <= 3 {
				continue
			} else {
				done <- struct{}{}
				break
			}
		}
		fmt.Printf("reps v = %v\n", resp.Response.Status)
	}
	time.Sleep(1 * time.Second)
	fmt.Println("Waiting")
}

func orChannelTest() {
	var or func(channels ...<-chan interface{}) <-chan interface{}
	or = func(channels ...<-chan interface{}) <-chan interface{} {
		switch len(channels) {
		case 0:
			return nil
		case 1:
			return channels[0]
		}

		orDone := make(chan interface{})
		go func() {
			defer close(orDone)
			defer fmt.Printf("close ordone:%+v\n", orDone)
			switch len(channels) {
			case 2:
				select {
				case <-channels[0]:
				case <-channels[1]:
				}
			default:
				select {
				case <-channels[0]:
				case <-channels[1]:
				case <-channels[2]:
				case <-or(append(channels[3:], orDone)...):
				}
			}
		}()
		return orDone
	}

	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}
	start := time.Now()

	<-or(sig(2*time.Hour), sig(1*time.Minute), sig(10*time.Second), sig(3*time.Second), sig(1*time.Second))
	fmt.Printf("done after %v\n", time.Since(start))
}

func testDoWriteWork() {
	newRandStream := func(done <-chan interface{}) <-chan int {
		randSream := make(chan int)
		go func() {
			defer fmt.Println("newRandStream closure exited.")
			defer close(randSream)
			// for {
			// 	select {
			// 	case <-done:
			// 		return
			// 	case randSream <- rand.Int():
			// 	}
			// }
			randSream <- rand.Int()
		}()
		return randSream
	}
	done := make(chan interface{})
	rand1 := newRandStream(done)
	for i := 1; i <= 3; i++ {
		fmt.Printf("%d: %d\n", i, <-rand1)
	}
	close(done)
	time.Sleep(1 * time.Second)
	fmt.Println("Done")
}
func testDowork() {
	dowork := func(done <-chan interface{}, str <-chan string) <-chan interface{} {
		completed := make(chan interface{})
		go func() {
			defer fmt.Println("dowork exited.")
			defer close(completed)
			for {
				select {
				case s := <-str:
					fmt.Println(s)
				case <-done:
					return
				}
			}
		}()
		return completed
	}

	done := make(chan interface{})
	ch := dowork(done, nil)
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Println("Canceling dowork go....")
		close(done)
	}()
	<-ch
	fmt.Println("Done.")
}

func testDefault() {
	done := make(chan interface{})
	go func() {
		time.Sleep(5 * time.Second)
		close(done)
	}()
	wcnt := 0

loop:
	for {
		select {
		case <-done:
			break loop
		default:
		}
		wcnt++
		time.Sleep(1 * time.Second)
	}
	fmt.Println("workcount ", wcnt)
}

func testtimeout() {
	var c1 chan int
	select {
	case <-c1:
	case <-time.After(1 * time.Minute):
		fmt.Println("timeout")
	}
}

func testallSelect() {
	c1 := make(chan interface{})
	c2 := make(chan interface{})
	close(c1)
	close(c2)
	c1count, c2count := 0, 0

	for i := 0; i < 1000; i++ {
		select {
		case <-c1:
			c1count++
		case <-c2:
			c2count++
		}
	}
	fmt.Println(c1count, c2count)
}

func testSelect() {
	start := time.Now()
	c := make(chan interface{})
	go func() {
		time.Sleep(1 * time.Second)
		close(c)
	}()

	fmt.Println("BLOCK on read")
	select {
	case <-c:
		fmt.Printf("Unblock %v later\n", time.Since(start))
	}
}

func channelOwnertest() {
	chanOwner := func() <-chan int {
		resultstream := make(chan int, 5)
		go func() {
			defer close(resultstream)
			for i := 0; i <= 5; i++ {
				resultstream <- i
			}
		}()
		return resultstream
	}

	resultstream := chanOwner()
	for result := range resultstream {
		fmt.Printf("received %d \n", result)
	}
	fmt.Println("Done!")
}

func testChannelread2w() {

	var ch chan interface{}
	close(ch)
}

func testChannel() {
	var stdoutbuff bytes.Buffer
	defer stdoutbuff.WriteTo(os.Stdout)

	stream := make(chan int, 4)
	go func() {
		defer close(stream)
		defer fmt.Fprintln(&stdoutbuff, "Produce One")
		for i := 0; i < 5; i++ {
			stream <- i
			fmt.Fprintf(&stdoutbuff, "Send : %d\n", i)
		}
	}()

	for i := 0; i < 5; i++ {
		k := <-stream
		fmt.Fprintf(&stdoutbuff, "Received %v\n", k)
	}
}
