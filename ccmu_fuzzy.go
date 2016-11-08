//CCMU Automatic Connector
//
//This tool provides automatic login feature for
//connect to the network authenticated by dr.com
//
//Author: ms2008vip@gmail.com at 2016/11/8 16:54:29
//

package main

import (
    "fmt"
    "net/http"
    "net/url"
    "os"
    "bufio"
    "io/ioutil"
    "strings"
    "math/rand"
    "time"
    l4g "github.com/alecthomas/log4go"
    "github.com/toqueteos/webbrowser"
)

const (
    URL = "http://192.168.161.2/"
    FILE = "ids.txt"
)

var log = make(l4g.Logger)
var buildstamp string = ""
var githash string = ""

func init() {
    log.AddFilter("stdout", l4g.DEBUG, l4g.NewConsoleLogWriter())
    log.AddFilter("file", l4g.ERROR, l4g.NewFileLogWriter("error.log", false))
}

func main() {
    defer log.Close()

    args := os.Args
    fmt.Println("os args:", os.Args)
    if len(args)==2 && (args[1]=="--version" || args[1] =="-v") {
        fmt.Printf("Git Commit Hash: %s\n", githash)
        fmt.Printf("UTC Build Time: %s\n", buildstamp)
        return
    }

    userList := fileTolines(FILE)
    rand.Seed(time.Now().Unix())
    //fmt.Println(userList)

    for {
        line := userList[rand.Intn(len(userList))]
        userInfo := strings.Split(line, "\t")
        account := userInfo[0]
        password := userInfo[3]
        //username := userInfo[1]

        //fmt.Println(account, password, username)
        //log.Info(line)
        if succ := checkPasswd(account, password); succ {
            balanceInfo := getBalance()
            fmt.Println(account, password, "Used Time:", balanceInfo[0], "Balance:", balanceInfo[1])

            if balanceInfo[0] >= 2400 && balanceInfo[1] == 0 {
                //fmt.Println(account, "time exceed!")
            } else {
                log.Info(account, password, "Used Time:", balanceInfo[0], "Balance:", balanceInfo[1])
                webbrowser.Open(URL)
                break
            }
        }
    }

    fmt.Println("看到这个，就是想证明下我不是个恶意程序，5s 之后就看不到我啦 :-)")
    <- time.After(5 * time.Second)
}

func fileTolines(filePath string) []string {
    f, err := os.Open(filePath)
    if err != nil {
        panic(err)
    }
    defer f.Close()

    var lines []string
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, err)
    }

    return lines
}

func chunkTolines(chunk string) []string {
    var lines []string
    lines = strings.Split(chunk, "\n")

    return lines
}

func checkPasswd(user, passwd string) bool {
	defer func() {
		if p := recover(); p != nil {
			err := p.(error)
			log.Critical("shit happens: %v [user: %s pass: %s]", err, user, passwd)
		}
	}()

    client := &http.Client{}

    // 构造认证请求
    form := url.Values{}
    form.Add("DDDDD", user)
    form.Add("upass", passwd)
    form.Add("0MKKey", "")

    req, err := http.NewRequest("POST", URL, strings.NewReader(form.Encode()))
    if err != nil {
        log.Error("failed to sent post request due to %s [user: %s pass: %s]", err, user, passwd)
        return false
    }

    // 防止被 dr.com banned
    req.Header.Set(`User-Agent`, `Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.2785.116 Safari/537.36`)
    req.Header.Set(`Referer`, URL)

    resp, err := client.Do(req)
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Error("failed to read login response body due to %s [user: %s pass: %s]", err, user, passwd)
        return false
    }
    //fmt.Println(string(body))

    // 判断认证是否通过
    line := chunkTolines(string(body))[5]
    if line[1] == "s" {
        // 认证通过
        return true
    } else if line[1] == "S" {
        return false
    } else {
        log.Warning("unexpected response! [user: %s pass: %s]", user, passwd)
        return false
    }
}

func getBalance() [2]int {
	defer func() {
		if p := recover(); p != nil {
			err := p.(error)
			log.Critical("shit happens: balance calc fauled %v", err)
		}
	}()

    balanceInfo := [2]int{0, 0}
    res, err := http.Get(URL)
    if err != nil {
        log.Error("failed to get balance info due to %s", err)
        return balanceInfo
    }

    result, err := ioutil.ReadAll(res.Body)
    defer res.Body.Close()
    if err != nil {
        log.Error("failed to read balance response body info due to %s", err)
        return balanceInfo
    }

    // 抓取已用时间
    line := chunkTolines(string(body))[6]
    re := regexp.MustCompile(`;time='(\d+) *';`)
    usedTime, _ := strconv.Atoi(re.FindStringSubmatch(line)[1])
    balanceInfo[0] = usedTime

    // 抓取余额
    re = regexp.MustCompile(`;fee='(\d+) *';`)
    fee, _ := strconv.Atoi(re.FindStringSubmatch(line)[1])
    amount := (fee - fee%100) / 10000
    balanceInfo[1] = amount

    return balanceInfo
}