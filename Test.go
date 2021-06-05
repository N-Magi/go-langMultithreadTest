package main

import (
	"bufio"
	"crypto/sha256"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"sync"
)

const intputfile string = "TESTDATA.txt"

const outfile string = "output.txt"

//処理スレッド数
const workerTh = 7

//Task構造体
type Task struct {
	Id  uint32 //順番管理用
	Res []byte //処理結果
}

//処理状態管理用構造体
type State struct {
	Start  bool
	Finish bool
}

func main() {
	fmt.Println("process Start with.. " + intputfile)
	//スレッド管理用waitgroup定義
	var wg sync.WaitGroup
	//file読み込み
	file, scnr := OpenFileWithScanner(intputfile)
	//スレッド数制御用channel
	sem := make(chan struct{}, workerTh)
	//返値用channel
	ch := make(chan Task, workerTh)
	//処理状態取得用channel
	state := make(chan State, 1) //{Start: false}
	//現在状態構造体を宣言してchannelに流す
	var ss = State{Start: false, Finish: false}
	state <- ss
	//出力用スレッドを定義、実行
	wg.Add(1)
	go func(statechan chan State) {
		var once sync.Once
		var buff []Task
		state := <-statechan

		for {
			//処理状態更新
			if len(statechan) >= 0 {
				state = <-statechan
			}
			//処理未開始
			if !state.Start && !state.Finish {
				fmt.Println("waiting")
				continue
			}
			//処理中
			if state.Start && !state.Finish {
				once.Do(func() { fmt.Print("reading") })
				fmt.Print(".")
				buff = append(buff, <-ch)
			}
			//処理終了時
			if state.Start && state.Finish {
				fmt.Println("finish!!")
				break
			}
		}
		//idをもとに正しい順番に並び替える
		sort.Slice(buff, func(i, j int) bool { return buff[i].Id < buff[j].Id })
		//サイズを定義する
		var outByte []byte
		for _, con := range buff {
			// //並び替え確認デバッグコード
			// fmt.Printf("out: %d\n\r", con.Id)
			outByte = append(outByte, con.Res...)
		}
		//出力用ファイルを作成、展開
		file, _ := os.Create(outfile)
		file.WriteString(hex.Dump(outByte))
		file.Close()
		wg.Done()
	}(state)

	//読み込み済み行数
	line := 0
	//スレッド数と読み込み行数に応じて新規処理を発注する
	for scnr.Scan() {
		sem <- struct{}{}
		wg.Add(1)
		go GetSHA256AsHexDump(scnr.Text(), uint32(line), ch, sem, &wg)
		line++
		//現在状態を出力
		ss = State{Start: true, Finish: false}
		state <- ss
	}
	//現在状態を終了にセット
	ss = State{Start: true, Finish: true}
	state <- ss
	//出力スレッドを待つ。
	wg.Wait()
	//閉じる
	file.Close()
}

//SHA256を計算し、task構造体としてchannel経由で返却する.
//str:入力文字列
//line:idこの場合は読み込み行数
//taskchan:返値用channel
//sem:スレッド数制御用channel
//wg:スレッド管理用waitgroup
func GetSHA256AsHexDump(str string, line uint32, taskchan chan Task, sem chan struct{}, wg *sync.WaitGroup) {
	//スレッド数管理用
	defer func() {
		<-sem
	}()
	res := sha256.Sum256([]byte(str))
	var task Task
	task.Res = res[:]
	task.Id = uint32(line)
	//  //マルチスレッド確認用のデバックコード
	// rand.Seed(time.Now().UnixNano())
	// time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	taskchan <- task
	wg.Done()
}

func OpenFileWithScanner(path string) (*os.File, *bufio.Scanner) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	scanner := bufio.NewScanner(file)
	return file, scanner
}
