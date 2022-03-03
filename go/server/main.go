package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	//"time"
	"os"
	"strconv"
	"unsafe"

	vosk "github.com/cyhkelvin/voskgo_rule/go"
)

type Message struct {
	Result []struct {
		Conf  float64
		End   float64
		Start float64
		Word  string
	}
	Text string
}

var m Message

func gen_wb_msg(res []byte) string {
	err := json.Unmarshal(res, &m)
	if err != nil {
		log.Fatal(err)
	}
	//t := time.Now().Format("2006-01-02 15:04:05")
	//msg := fmt.Sprintf("final %s %s", string(m.Text[:]), t)
	//return msg
	return string(m.Text[:])
}

func check(err error, msg string) bool {
	if err != nil {
		log.Println(msg)
		log.Println(err)
		return true
	}
	return false
}

func RecognizeRoutine(model *vosk.VoskModel, sampleRate float64) {

}

func main() {
	// init parameters
	Port := fmt.Sprintf(":%s", os.Args[1])
	sampleRate, err := strconv.ParseFloat(os.Args[2], 64)
	if check(err, "[asr-server] error: sample rate setting wrong! use default: 8000.0") {
		sampleRate = 8000.0
	}
	// initial model and recognizer
	model, err := vosk.NewModel("model")
	_ = check(err, "[asr-server] error: model loading error!")

	upgrader := &websocket.Upgrader{
		//如果有 cross domain 的需求，可加入這個，不檢查 cross domain
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if check(err, "[asr-server] upgrade error") {
			return
		}
		defer func() {
			log.Println("disconnect !!")
			c.Close()
		}()

		rec, err := vosk.NewRecognizer(model, sampleRate)
		_ = check(err, "[asr-server] error: recognizer init error!")
		rec.SetWords(1)

		params := r.URL.Query()
		session_id := params.Get("id")
		log.Printf("receive: %s\n", session_id)
		for {
			mtype, msg, err := c.ReadMessage()
			if check(err, "[asr-server] read error!") {
				break
			}
			if rec.AcceptWaveform(msg) != 0 {
				wb_msg := gen_wb_msg(rec.Result())
				log.Printf("reconize: %s\n", wb_msg)
				err = c.WriteMessage(mtype, []byte(wb_msg))
				_ = check(err, "[asr-server] write error.")
				break
			}
			log.Printf("receive size: %d\n", unsafe.Sizeof(msg))

			if bytes.Equal(msg, []byte("{\"eof\" : 1}")) {
				break
			}
		}
		res_msg := gen_wb_msg(rec.FinalResult())
		err = c.WriteMessage(1, []byte(res_msg))
		_ = check(err, "[asr-server] write error!")
		log.Println("result:", res_msg)

		err = c.WriteMessage(1, []byte("{\"msg\": \"ENDDD\"}"))
		_ = check(err, "[asr-server] end error")
	})
	log.Println("server start at ", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}
