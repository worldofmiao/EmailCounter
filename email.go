// email counter
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/mxk/go-imap/imap"
)

type requestBody struct {
	Login    string
	Password string
	From     string
}

type responseBody struct {
	Count string
}

func main() {
	http.HandleFunc("/getEmail", emailService)
	http.ListenAndServe(":8082", nil)
}

//count the number of emails in a mailbox from a certain sender
func countEmails(login string, password string, from string) (string, error) {
	var (
		c        *imap.Client
		cmd      *imap.Command
		rsp      *imap.Response
		count    int
		newError error
	)
	count = 0
	// Connect to the server
	if strings.Contains(login, "gmail.com") {
		c, _ = imap.DialTLS("imap.gmail.com", nil)
	} else if strings.Contains(login, "hotmail.com") {
		c, _ = imap.DialTLS("imap-mail.outlook.com", nil)
	} else if strings.Contains(login, "comcast.net") {
		c, _ = imap.DialTLS("imap.comcast.net", nil)
	} else {
		newError = errors.New("Unsupported Email Address")
		return strconv.Itoa(count), newError
	}

	// Remember to log out and close the connection when finished
	defer c.Logout(30 * time.Second)

	// Authenticate
	//if c.State() == imap.Login {
	cmd, err := c.Login(login, password)
	if err == nil {
		// Open a mailbox (synchronous command - no need for imap.Wait)
		c.Select("INBOX", true)

		// Fetch the headers of the all messages
		set, _ := imap.NewSeqSet("")
		set.AddRange(1, c.Mailbox.Messages)
		cmd, _ = c.Fetch(set, "RFC822.HEADER")

		// Process responses while the command is running
		for cmd.InProgress() {
			// Wait for the next response (no timeout)
			c.Recv(-1)

			// Process command data
			for _, rsp = range cmd.Data {
				header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
				if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
					if strings.Contains(msg.Header.Get("From"), from) {
						count++
					}
				}
			}
			cmd.Data = nil
			c.Data = nil
		}

		// Check command completion status
		if rsp, err := cmd.Result(imap.OK); err != nil {
			if err == imap.ErrAborted {
				fmt.Println("Fetch command aborted")
			} else {
				fmt.Println("Fetch error:", rsp.Info)
			}
		}
		newError = nil
		//c.Logout(30 * time.Second)
	} else {
		newError = errors.New(err.Error())
		return strconv.Itoa(count), newError
	}

	return strconv.Itoa(count), newError
}

//api handler
func emailService(w http.ResponseWriter, r *http.Request) {
	//check http request type
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Request"))
		return
	}
	decoder := json.NewDecoder(r.Body)
	var req requestBody
	err := decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Json"))
		return
	}
	//check request json
	if req.From != "" && req.Login != "" && req.Password != "" {
		count, someError := countEmails(req.Login, req.Password, req.From)
		if someError == nil {
			resp := responseBody{Count: count}
			json.NewEncoder(w).Encode(resp)
			return
		} else if someError.Error() == "Unsupported Email Address" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(someError.Error()))
			return
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(someError.Error()))
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("some fields are missing"))
		return
	}
}
