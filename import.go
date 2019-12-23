//
// Copyright (c) 2019 Ted Unangst <tedu@tedunangst.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"humungus.tedunangst.com/r/webs/htfilter"
)

func importMain(username, flavor, source string) {
	switch flavor {
	case "mastodon":
		importMastodon(username, source)
	case "twitter":
		importTwitter(username, source)
	default:
		log.Fatal("unknown source flavor")
	}
}

func importMastodon(username, source string) {
	user, err := butwhatabout(username)
	if err != nil {
		log.Fatal(err)
	}
	type Toot struct {
		Id           string
		Type         string
		To           []string
		Cc           []string
		Summary      string
		Content      string
		InReplyTo    string
		Conversation string
		Published    time.Time
		Tag          []struct {
			Type string
			Name string
		}
		Attachment []struct {
			Type      string
			MediaType string
			Url       string
			Name      string
		}
	}
	var outbox struct {
		OrderedItems []struct {
			Object Toot
		}
	}
	fd, err := os.Open(source + "/outbox.json")
	if err != nil {
		log.Fatal(err)
	}
	dec := json.NewDecoder(fd)
	err = dec.Decode(&outbox)
	if err != nil {
		log.Fatalf("error parsing json: %s", err)
	}
	fd.Close()

	havetoot := func(xid string) bool {
		var id int64
		row := stmtFindXonk.QueryRow(user.ID, xid)
		err := row.Scan(&id)
		if err == nil {
			return true
		}
		return false
	}

	re_tootid := regexp.MustCompile("[^/]+$")
	for _, item := range outbox.OrderedItems {
		toot := item.Object
		tootid := re_tootid.FindString(toot.Id)
		xid := fmt.Sprintf("%s/%s/%s", user.URL, honkSep, tootid)
		if havetoot(xid) {
			continue
		}
		honk := Honk{
			UserID:   user.ID,
			What:     "honk",
			Honker:   user.URL,
			XID:      xid,
			RID:      toot.InReplyTo,
			Date:     toot.Published,
			URL:      xid,
			Audience: append(toot.To, toot.Cc...),
			Noise:    toot.Content,
			Convoy:   toot.Conversation,
			Whofore:  2,
			Format:   "html",
			Precis:   toot.Summary,
		}
		if honk.RID != "" {
			honk.What = "tonk"
		}
		if !loudandproud(honk.Audience) {
			honk.Whofore = 3
		}
		for _, att := range toot.Attachment {
			switch att.Type {
			case "Document":
				fname := fmt.Sprintf("%s/%s", source, att.Url)
				data, err := ioutil.ReadFile(fname)
				if err != nil {
					log.Printf("error reading media: %s", fname)
					continue
				}
				u := xfiltrate()
				name := att.Name
				desc := name
				newurl := fmt.Sprintf("https://%s/d/%s", serverName, u)
				fileid, err := savefile(u, name, desc, newurl, att.MediaType, true, data)
				if err != nil {
					log.Printf("error saving media: %s", fname)
					continue
				}
				donk := &Donk{
					FileID: fileid,
				}
				honk.Donks = append(honk.Donks, donk)
			}
		}
		for _, t := range toot.Tag {
			switch t.Type {
			case "Hashtag":
				honk.Onts = append(honk.Onts, t.Name)
			}
		}
		savehonk(&honk)
	}
}

func importTwitter(username, source string) {
	user, err := butwhatabout(username)
	if err != nil {
		log.Fatal(err)
	}

	type Tweet struct {
		ID_str                  string
		Created_at              string
		Full_text               string
		In_reply_to_screen_name string
		In_reply_to_status_id   string
		Entities                struct {
			Hashtags []struct {
				Text string
			}
			Media []struct {
				Url       string
				Media_url string
			}
			Urls []struct {
				Url          string
				Expanded_url string
			}
		}
		date   time.Time
		convoy string
	}

	var tweets []*Tweet
	fd, err := os.Open(source + "/tweet.js")
	if err != nil {
		log.Fatal(err)
	}
	// skip past window.YTD.tweet.part0 =
	fd.Seek(25, 0)
	dec := json.NewDecoder(fd)
	err = dec.Decode(&tweets)
	if err != nil {
		log.Fatalf("error parsing json: %s", err)
	}
	fd.Close()
	tweetmap := make(map[string]*Tweet)
	for _, t := range tweets {
		t.date, _ = time.Parse("Mon Jan 02 15:04:05 -0700 2006", t.Created_at)
		tweetmap[t.ID_str] = t
	}
	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].date.Before(tweets[j].date)
	})
	havetwid := func(xid string) bool {
		var id int64
		row := stmtFindXonk.QueryRow(user.ID, xid)
		err := row.Scan(&id)
		if err == nil {
			return true
		}
		return false
	}

	for _, t := range tweets {
		xid := fmt.Sprintf("%s/%s/%s", user.URL, honkSep, t.ID_str)
		if havetwid(xid) {
			continue
		}
		what := "honk"
		noise := ""
		if parent := tweetmap[t.In_reply_to_status_id]; parent != nil {
			t.convoy = parent.convoy
			what = "tonk"
		} else {
			t.convoy = "data:,acoustichonkytonk-" + t.ID_str
			if t.In_reply_to_screen_name != "" {
				noise = fmt.Sprintf("re: https://twitter.com/%s/status/%s\n\n",
					t.In_reply_to_screen_name, t.In_reply_to_status_id)
				what = "tonk"
			}
		}
		audience := []string{thewholeworld}
		honk := Honk{
			UserID:   user.ID,
			Username: user.Name,
			What:     what,
			Honker:   user.URL,
			XID:      xid,
			Date:     t.date,
			Format:   "markdown",
			Audience: audience,
			Convoy:   t.convoy,
			Public:   true,
			Whofore:  2,
		}
		noise += t.Full_text
		// unbelievable
		noise = html.UnescapeString(noise)
		for _, r := range t.Entities.Urls {
			noise = strings.Replace(noise, r.Url, r.Expanded_url, -1)
		}
		for _, m := range t.Entities.Media {
			u := m.Media_url
			idx := strings.LastIndexByte(u, '/')
			u = u[idx+1:]
			fname := fmt.Sprintf("%s/tweet_media/%s-%s", source, t.ID_str, u)
			data, err := ioutil.ReadFile(fname)
			if err != nil {
				log.Printf("error reading media: %s", fname)
				continue
			}
			newurl := fmt.Sprintf("https://%s/d/%s", serverName, u)

			fileid, err := savefile(u, u, u, newurl, "image/jpg", true, data)
			if err != nil {
				log.Printf("error saving media: %s", fname)
				continue
			}
			donk := &Donk{
				FileID: fileid,
			}
			honk.Donks = append(honk.Donks, donk)
			noise = strings.Replace(noise, m.Url, "", -1)
		}
		for _, ht := range t.Entities.Hashtags {
			honk.Onts = append(honk.Onts, "#"+ht.Text)
		}
		honk.Noise = noise
		savehonk(&honk)
	}
}
