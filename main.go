package main

import (
	"encoding/json"
	"flag"
	"fmt"
	badger "github.com/dgraph-io/badger"
	"github.com/mattn/go-xmpp"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)
import "net/http"

const Url = "https://miapi.pandorabots.com/talk"

const NickName = "Mitsuku"

const Room = "test-mitsuku"

type SendResponse struct {
	Responses []string
	Sessionid int
	Channel   int
}

func getClientName() string {
	nsec := time.Now().UnixNano() / 1000000
	return "cw" + fmt.Sprintf("%x", nsec)
}

func send(msg string, sessionId int, clientName string) ([]string, int, error) {
	data := url.Values{
		"input":       {msg},
		"channel":     {"6"},
		"botkey":      {"n0M6dW2XZacnOgCWTp0FRYUuMjSfCkJGgobNpgPv9060_72eKnu3Yl-o1v2nFGtSXqfwJBG2Ros~"},
		"client_name": {clientName},
	}

	if sessionId != -1 {
		data.Add("sessionid", strconv.Itoa(sessionId))
	}

	req, err := http.NewRequest("POST", Url, strings.NewReader(data.Encode()))

	if err != nil {
		return []string{}, -1, err
	}

	req.Header.Add("Referer", "https://www.pandorabots.com/mitsuku/")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return []string{}, -1, err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return []string{}, -1, err
	}

	var parsed SendResponse

	if json.Unmarshal(bytes, &parsed) != nil {
		return []string{}, -1, err
	}

	return parsed.Responses, parsed.Sessionid, nil
}

func getResource(jid string) (string, string) {
	split := strings.Split(jid, "/")
	if len(split) == 2 {
		return split[0], split[1]
	} else {
		return "", ""
	}
}

func openDb() (*badger.DB, error) {
	return badger.Open(badger.DefaultOptions("/tmp/mitsuku-xmpp"))
}

func storeResourceClientName(db *badger.DB, resource string, clientName string) error {
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(resource), []byte(clientName))
		return err
	})
	return err
}

func getResourceClientName(db *badger.DB, resource string, cb func(clientName string)) error {
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(resource))

		if err == badger.ErrKeyNotFound {
			cb("")
			return nil
		}

		if err != nil {
			return err
		}
		stored, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		cb(string(stored))
		return nil
	})
	return err
}

func main() {
	var password = flag.String("password", "", "XMPP password for user")
	flag.Parse()

	db, err := openDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	options := xmpp.Options{
		Host:          "chat.codingame.com:5222",
		User:          "3774175@chat.codingame.com",
		Password:      *password,
		NoTLS:         true,
		Debug:         true,
		Session:       true,
		Status:        "xa",
		StatusMessage: "Not connected",
	}

	talk, err := options.NewClient()

	if err != nil {
		log.Fatal(err)
	}

	_, err = talk.JoinMUCNoHistory(Room+"@conference.codingame.com", NickName)

	if err != nil {
		log.Fatal(err)
	}

	var _ int = -1
	var apiResponses []string
	for {
		chat, err := talk.Recv()
		if err != nil {
			log.Fatal(err)
		}
		switch v := chat.(type) {
		case xmpp.Chat:
			_, myResource := getResource(talk.JID())
			otherBareJid, otherResource := getResource(v.Remote)
			if otherResource != "" && otherResource != myResource && otherResource != NickName && strings.Contains(v.Text, NickName) {
				fmt.Println("Received chat:", v.Remote, v.Text)
				fmt.Println("not me, I'm", myResource, "message is from", otherResource)
				fmt.Println(v)

				err := getResourceClientName(db, otherResource, func(clientName string) {
					if clientName == "" {
						clientName = getClientName()
						err := storeResourceClientName(db, otherResource, clientName)
						if err != nil {
							fmt.Println("Error when storing client name", err)
							return
						}
					}

					fmt.Println("client name is", clientName)

					apiResponses, _, err = send(v.Text, -1, clientName)
					if err != nil {
						fmt.Println("Error calling Mitsuku api", err)
					}

					for _, r := range apiResponses {
						_, err := talk.Send(xmpp.Chat{
							Remote: otherBareJid,
							Type:   "groupchat",
							Text:   r,
						})

						if err != nil {
							fmt.Println("Error sending message", err)
						}
					}
				})
				if err != nil {
					fmt.Println("Error fetching client name", err)
				}
			}
		case xmpp.Presence:
			fmt.Println("presence:", v.From, v.Show, v.Status)
		}
	}
}
