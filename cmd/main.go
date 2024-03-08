package main

import (
	"flag"
	"fmt"
	"kubefasthdfs/rocksdb"
	"os"
	"runtime"
)

var configVersion = flag.Bool("v", false, "show version")
var (
	Version    string
	CommitID   string
	BranchName string
	BuildTime  string
)

func DumpVersion() string {
	return fmt.Sprintf("kubefasthdfs\n"+
		"Version : %s\n"+
		"Branch  : %s\n"+
		"Commit  : %s\n"+
		"Build   : %s %s %s %s\n",
		Version,
		BranchName,
		CommitID,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, BuildTime)
}

func main() {
	flag.Parse()
	version := DumpVersion()
	if *configVersion {
		fmt.Printf("%v", version)
	}

	db, err := rocksdb.NewRocksDBStore("/tmp/rocksdbdata", 64*1024, 8*1024)
	if err != nil {
		fmt.Printf("open db error %v", err)
		os.Exit(1)
	}

	result, err := db.Put("abc", []byte("hello world1"), false)
	if err != nil {
		fmt.Printf("write data to db error %v", err)
		os.Exit(1)
	}
	fmt.Printf("add result is key=%v, value=%v\n", "abc", string(result.([]byte)))

	result, err = db.Get("abc")
	if err != nil {
		fmt.Printf("read data from db error %v", err)
		os.Exit(1)
	}
	fmt.Printf("get result is key=%v, value=%v\n", "abc", string(result.([]byte)))

	result, err = db.Del("abc", false)
	if err != nil {
		fmt.Printf("delete data from db error %v", err)
		os.Exit(1)
	}
	fmt.Printf("delete result is key=%v, value=%v\n", "abc", string(result.([]byte)))
}
