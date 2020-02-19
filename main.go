package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/poofyleek/glog"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var version = "v0.0.1"

type modifiedFile struct {
	Name  string
	IsDir bool
}

type lastcheckInfo struct {
	LastcheckTime int64
}

var bindAddr, srcURI string

func main() {
	flag.StringVar(&bindAddr, "server", "",
		"run as server listening at ip:port")
	flag.StringVar(&srcURI, "from", "",
		"run as client pulling data from server at URI")
	flag.Parse()
	if bindAddr == "" && srcURI == "" {
		fmt.Printf("Must be a server or a client. Use -server or -from.\n")
		flag.Usage()
		os.Exit(1)
	}
	if bindAddr != "" && srcURI != "" {
		glog.Fatalf("can't be server and client at the same time")
	}
	if bindAddr != "" {
		dirSyncServer()
		os.Exit(0)
	}
	if srcURI != "" {
		dirSyncClient()
	}
	os.Exit(0)
}

func dirSyncServer() {
	http.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			glog.V(4).Infof("%v", r.URL.Path)
			if r.URL.Path == "/" {
				res, err := scanDir(".")
				if err != nil {
					err = fmt.Errorf("can't scan directory, %v", err)
					http.Error(w, err.Error(),
						http.StatusInternalServerError)
					glog.Error(err)
					return
				}
				rjson, _ := json.Marshal(res)
				jstr := string(rjson)
				fmt.Fprint(w, jstr)
				return
			}
			http.ServeFile(w, r, r.URL.Path[1:])
		})
	glog.Infof("HTTP server @ %v", bindAddr)
	log.Fatal(http.ListenAndServe(bindAddr, nil))
}

func scanDir(root string) (*[]modifiedFile, error) {
	lastcheckPath := root + "/lastcheck.json"
	fmt.Printf("checking path %s\n", lastcheckPath)
	lastcheck, err := ioutil.ReadFile(lastcheckPath)
	lastcheckTime := int64(0)
	if err != nil {
		glog.V(4).Infof("no last check file")
	} else {
		glog.V(4).Infof("last check: %s\n", lastcheck)
		lc := lastcheckInfo{}
		err := json.Unmarshal([]byte(lastcheck), &lc)
		if err != nil {
			return nil, err
		}
		lastcheckTime = lc.LastcheckTime
	}
	glog.V(4).Infof("lastcheckTime: %d\n", lastcheckTime)
	modList := make([]modifiedFile, 0)
	cb := func(path string, fi os.FileInfo, err error) error {
		t := fi.ModTime().Unix()
		if t > lastcheckTime {
			glog.V(4).Infof("found: %s modTime %d > %d\n", path, t, lastcheckTime)
			isdir := false
			if fi.IsDir() {
				isdir = true
			}
			mf := modifiedFile{path, isdir}
			modList = append(modList, mf)
		}
		return nil
	}
	filepath.Walk(root, cb)
	lastcheckTime = time.Now().Unix()
	err = ioutil.WriteFile(lastcheckPath, []byte(fmt.Sprintf(`{"LastcheckTime": %d}`, lastcheckTime)), 0666)
	if err != nil {
		return nil, err
	}
	glog.V(4).Infof("update lastcheckTime to %d\n", lastcheckTime)
	glog.V(4).Infof("modList %v\n", modList)
	return &modList, nil
}

func dirSyncClient() {
	resp, err := http.Get(srcURI)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var modList []modifiedFile
	err = decoder.Decode(&modList)
	if err != nil {
		glog.Errorf("can't decode body")
		return
	}
	glog.V(4).Infof("got %v", modList)
	for _, f := range modList {
		if f.Name == "." {
			continue
		}
		if f.IsDir {
			err = os.MkdirAll(f.Name, 0777)
			if err != nil {
				glog.Error(err)
			}
			continue
		}
		saveFile(f.Name)
	}
}

func saveFile(name string) {
	resp, err := http.Get(srcURI + "/" + name)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	glog.V(4).Info("creating ", name)
	out, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(out, resp.Body)
}
