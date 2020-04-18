package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)
import "net/http"

const Url = "https://miapi.pandorabots.com/talk"

type SendResponse struct {
	Responses []string
	Sessionid int
	Channel   int
}

func send(msg string, sessionId int) (*string, *int, error) {
	data := url.Values{
		"input":   {msg},
		"channel": {"6"},
		"botkey":  {"n0M6dW2XZacnOgCWTp0FRYUuMjSfCkJGgobNpgPv9060_72eKnu3Yl-o1v2nFGtSXqfwJBG2Ros~"},
	}

	if sessionId != -1 {
		data.Add("sessionid", strconv.Itoa(sessionId))
	}

	req, err := http.NewRequest("POST", Url, strings.NewReader(data.Encode()))

	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("Referer", "https://www.pandorabots.com/mitsuku/")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, nil, err
	}

	var parsed SendResponse

	if json.Unmarshal(bytes, &parsed) != nil {
		return nil, nil, err
	}

	responsesJoined := strings.Join(parsed.Responses, "\n")
	return &responsesJoined, &parsed.Sessionid, nil
}

func main() {
	response, sessionId, err := send("Hey my name is Yann", -1)
	if err != nil {
		panic(err)
	}

	fmt.Println("got response: <", *response, "> with sessionId", *sessionId)

	response2, sessionId2, err := send("What is my name?", *sessionId)
	if err != nil {
		panic(err)
	}

	fmt.Println("got response: <", *response2, "> with sessionId", *sessionId2)
}
