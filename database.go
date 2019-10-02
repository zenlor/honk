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
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"bytes"
	"encoding/json"

	"humungus.tedunangst.com/r/webs/login"
)

func butwhatabout(name string) (*WhatAbout, error) {
	row := stmtWhatAbout.QueryRow(name)
	var user WhatAbout
	var options string
	err := row.Scan(&user.ID, &user.Name, &user.Display, &user.About, &user.Key, &options)
	user.URL = fmt.Sprintf("https://%s/%s/%s", serverName, userSep, user.Name)
	user.SkinnyCSS = strings.Contains(options, " skinny ")
	return &user, err
}

func gethonkers(userid int64) []*Honker {
	rows, err := stmtHonkers.Query(userid)
	if err != nil {
		log.Printf("error querying honkers: %s", err)
		return nil
	}
	defer rows.Close()
	var honkers []*Honker
	for rows.Next() {
		var f Honker
		var combos string
		err = rows.Scan(&f.ID, &f.UserID, &f.Name, &f.XID, &f.Flavor, &combos)
		f.Combos = strings.Split(strings.TrimSpace(combos), " ")
		if err != nil {
			log.Printf("error scanning honker: %s", err)
			return nil
		}
		honkers = append(honkers, &f)
	}
	return honkers
}

func getdubs(userid int64) []*Honker {
	rows, err := stmtDubbers.Query(userid)
	if err != nil {
		log.Printf("error querying dubs: %s", err)
		return nil
	}
	defer rows.Close()
	var honkers []*Honker
	for rows.Next() {
		var f Honker
		err = rows.Scan(&f.ID, &f.UserID, &f.Name, &f.XID, &f.Flavor)
		if err != nil {
			log.Printf("error scanning honker: %s", err)
			return nil
		}
		honkers = append(honkers, &f)
	}
	return honkers
}

func allusers() []login.UserInfo {
	var users []login.UserInfo
	rows, _ := opendatabase().Query("select userid, username from users")
	defer rows.Close()
	for rows.Next() {
		var u login.UserInfo
		rows.Scan(&u.UserID, &u.Username)
		users = append(users, u)
	}
	return users
}

func getxonk(userid int64, xid string) *Honk {
	row := stmtOneXonk.QueryRow(userid, xid)
	return scanhonk(row)
}

func getbonk(userid int64, xid string) *Honk {
	row := stmtOneBonk.QueryRow(userid, xid)
	return scanhonk(row)
}

func getpublichonks() []*Honk {
	dt := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(dbtimeformat)
	rows, err := stmtPublicHonks.Query(dt)
	return getsomehonks(rows, err)
}
func gethonksbyuser(name string, includeprivate bool) []*Honk {
	dt := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(dbtimeformat)
	whofore := 2
	if includeprivate {
		whofore = 3
	}
	rows, err := stmtUserHonks.Query(whofore, name, dt)
	return getsomehonks(rows, err)
}
func gethonksforuser(userid int64) []*Honk {
	dt := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(dbtimeformat)
	rows, err := stmtHonksForUser.Query(userid, dt, userid, userid)
	return getsomehonks(rows, err)
}
func gethonksforuserfirstclass(userid int64) []*Honk {
	dt := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(dbtimeformat)
	rows, err := stmtHonksForUserFirstClass.Query(userid, dt, userid, userid)
	return getsomehonks(rows, err)
}
func gethonksforme(userid int64) []*Honk {
	dt := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(dbtimeformat)
	rows, err := stmtHonksForMe.Query(userid, dt, userid)
	return getsomehonks(rows, err)
}
func gethonksbyhonker(userid int64, honker string) []*Honk {
	rows, err := stmtHonksByHonker.Query(userid, honker, userid)
	return getsomehonks(rows, err)
}
func gethonksbyxonker(userid int64, xonker string) []*Honk {
	rows, err := stmtHonksByXonker.Query(userid, xonker, xonker, userid)
	return getsomehonks(rows, err)
}
func gethonksbycombo(userid int64, combo string) []*Honk {
	combo = "% " + combo + " %"
	rows, err := stmtHonksByCombo.Query(userid, combo, userid)
	return getsomehonks(rows, err)
}
func gethonksbyconvoy(userid int64, convoy string) []*Honk {
	rows, err := stmtHonksByConvoy.Query(userid, userid, convoy)
	honks := getsomehonks(rows, err)
	for i, j := 0, len(honks)-1; i < j; i, j = i+1, j-1 {
		honks[i], honks[j] = honks[j], honks[i]
	}
	return honks
}
func gethonksbysearch(userid int64, q string) []*Honk {
	q = "%" + q + "%"
	rows, err := stmtHonksBySearch.Query(userid, q)
	honks := getsomehonks(rows, err)
	return honks
}
func gethonksbyontology(userid int64, name string) []*Honk {
	rows, err := stmtHonksByOntology.Query(name, userid, userid)
	honks := getsomehonks(rows, err)
	return honks
}

func getsomehonks(rows *sql.Rows, err error) []*Honk {
	if err != nil {
		log.Printf("error querying honks: %s", err)
		return nil
	}
	defer rows.Close()
	var honks []*Honk
	for rows.Next() {
		h := scanhonk(rows)
		if h != nil {
			honks = append(honks, h)
		}
	}
	rows.Close()
	donksforhonks(honks)
	return honks
}

type RowLike interface {
	Scan(dest ...interface{}) error
}

func scanhonk(row RowLike) *Honk {
	h := new(Honk)
	var dt, aud string
	err := row.Scan(&h.ID, &h.UserID, &h.Username, &h.What, &h.Honker, &h.Oonker, &h.XID, &h.RID,
		&dt, &h.URL, &aud, &h.Noise, &h.Precis, &h.Format, &h.Convoy, &h.Whofore, &h.Flags)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("error scanning honk: %s", err)
		}
		return nil
	}
	h.Date, _ = time.Parse(dbtimeformat, dt)
	h.Audience = strings.Split(aud, " ")
	h.Public = !keepitquiet(h.Audience)
	return h
}

func donksforhonks(honks []*Honk) {
	db := opendatabase()
	var ids []string
	hmap := make(map[int64]*Honk)
	for _, h := range honks {
		ids = append(ids, fmt.Sprintf("%d", h.ID))
		hmap[h.ID] = h
	}
	// grab donks
	q := fmt.Sprintf("select honkid, donks.fileid, xid, name, description, url, media, local from donks join filemeta on donks.fileid = filemeta.fileid where honkid in (%s)", strings.Join(ids, ","))
	rows, err := db.Query(q)
	if err != nil {
		log.Printf("error querying donks: %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var hid int64
		var d Donk
		err = rows.Scan(&hid, &d.FileID, &d.XID, &d.Name, &d.Desc, &d.URL, &d.Media, &d.Local)
		if err != nil {
			log.Printf("error scanning donk: %s", err)
			continue
		}
		h := hmap[hid]
		h.Donks = append(h.Donks, &d)
	}
	rows.Close()

	// grab onts
	q = fmt.Sprintf("select honkid, ontology from onts where honkid in (%s)", strings.Join(ids, ","))
	rows, err = db.Query(q)
	if err != nil {
		log.Printf("error querying onts: %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var hid int64
		var o string
		err = rows.Scan(&hid, &o)
		if err != nil {
			log.Printf("error scanning donk: %s", err)
			continue
		}
		h := hmap[hid]
		h.Onts = append(h.Onts, o)
	}
	rows.Close()
	// grab meta
	q = fmt.Sprintf("select honkid, genus, json from honkmeta where honkid in (%s)", strings.Join(ids, ","))
	rows, err = db.Query(q)
	if err != nil {
		log.Printf("error querying honkmeta: %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var hid int64
		var genus, j string
		err = rows.Scan(&hid, &genus, &j)
		if err != nil {
			log.Printf("error scanning honkmeta: %s", err)
			continue
		}
		h := hmap[hid]
		switch genus {
		case "place":
			p := new(Place)
			err = unjsonify(j, p)
			if err != nil {
				log.Printf("error parsing place: %s", err)
				continue
			}
			h.Place = p
		case "oldrev":
		default:
			log.Printf("unknown meta genus: %s", genus)
		}
	}
	rows.Close()
}

func savefile(xid string, name string, desc string, url string, media string, local bool, data []byte) (int64, error) {
	res, err := stmtSaveFile.Exec(xid, name, desc, url, media, local)
	if err != nil {
		return 0, err
	}
	fileid, _ := res.LastInsertId()
	_, err = stmtSaveFileData.Exec(xid, media, data)
	if err != nil {
		return 0, err
	}
	return fileid, nil
}

func savehonk(h *Honk) error {
	dt := h.Date.UTC().Format(dbtimeformat)
	aud := strings.Join(h.Audience, " ")

	res, err := stmtSaveHonk.Exec(h.UserID, h.What, h.Honker, h.XID, h.RID, dt, h.URL,
		aud, h.Noise, h.Convoy, h.Whofore, h.Format, h.Precis,
		h.Oonker, h.Flags)
	if err != nil {
		log.Printf("err saving honk: %s", err)
		return err
	}
	h.ID, _ = res.LastInsertId()
	err = saveextras(h)
	return err
}

func saveextras(h *Honk) error {
	for _, d := range h.Donks {
		_, err := stmtSaveDonk.Exec(h.ID, d.FileID)
		if err != nil {
			log.Printf("err saving donk: %s", err)
			return err
		}
	}
	for _, o := range h.Onts {
		_, err := stmtSaveOnt.Exec(strings.ToLower(o), h.ID)
		if err != nil {
			log.Printf("error saving ont: %s", err)
			return err
		}
	}
	if p := h.Place; p != nil {
		j, err := jsonify(p)
		if err != nil {
			_, err = stmtSaveMeta.Exec(h.ID, "genus", j)
		}
		if err != nil {
			log.Printf("error saving place: %s", err)
			return err
		}
	}

	return nil
}

func deleteextras(honkid int64) {
	_, err := stmtDeleteDonks.Exec(honkid)
	if err != nil {
		log.Printf("error deleting: %s", err)
	}
	_, err = stmtDeleteOnts.Exec(honkid)
	if err != nil {
		log.Printf("error deleting: %s", err)
	}
	_, err = stmtDeleteMeta.Exec(honkid)
	if err != nil {
		log.Printf("error deleting: %s", err)
	}
}

func deletehonk(honkid int64) {
	deleteextras(honkid)
	_, err := stmtDeleteHonk.Exec(honkid)
	if err != nil {
		log.Printf("error deleting: %s", err)
	}
}

func jsonify(what interface{}) (string, error) {
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	e.SetIndent("", "")
	err := e.Encode(what)
	return buf.String(), err
}

func unjsonify(s string, dest interface{}) error {
	d := json.NewDecoder(strings.NewReader(s))
	err := d.Decode(dest)
	return err
}

func updatehonk(h *Honk) {
	old := getxonk(h.UserID, h.XID)
	oldrev := OldRevision{Precis: old.Precis, Noise: old.Noise}

	deleteextras(h.ID)

	dt := h.Date.UTC().Format(dbtimeformat)
	stmtUpdateHonk.Exec(h.Precis, h.Noise, h.Format, dt, h.ID)

	saveextras(h)
	j, err := jsonify(&oldrev)
	if err != nil {
		log.Printf("error jsonify oldrev: %s", err)
		return
	}
	_, err = stmtSaveMeta.Exec(old.ID, "oldrev", j)
	if err != nil {
		log.Printf("error saving oldrev: %s", err)
		return
	}
}

func cleanupdb(arg string) {
	db := opendatabase()
	days, err := strconv.Atoi(arg)
	var sqlargs []interface{}
	var where string
	if err != nil {
		honker := arg
		expdate := time.Now().UTC().Add(-3 * 24 * time.Hour).Format(dbtimeformat)
		where = "dt < ? and whofore = 0 and honker = ?"
		sqlargs = append(sqlargs, expdate)
		sqlargs = append(sqlargs, honker)
	} else {
		expdate := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(dbtimeformat)
		where = "dt < ? and whofore = 0 and convoy not in (select convoy from honks where whofore = 2 or whofore = 3)"
		sqlargs = append(sqlargs, expdate)
	}
	doordie(db, "delete from honks where "+where, sqlargs...)
	doordie(db, "delete from donks where honkid not in (select honkid from honks)")
	doordie(db, "delete from onts where honkid not in (select honkid from honks)")
	doordie(db, "delete from honkmeta where honkid not in (select honkid from honks)")

	doordie(db, "delete from filemeta where fileid not in (select fileid from donks)")
	for _, u := range allusers() {
		doordie(db, "delete from zonkers where userid = ? and wherefore = 'zonvoy' and zonkerid < (select zonkerid from zonkers where userid = ? and wherefore = 'zonvoy' order by zonkerid desc limit 1 offset 200)", u.UserID, u.UserID)
	}

	filexids := make(map[string]bool)
	blobdb := openblobdb()
	rows, err := blobdb.Query("select xid from filedata")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var xid string
		err = rows.Scan(&xid)
		if err != nil {
			log.Fatal(err)
		}
		filexids[xid] = true
	}
	rows.Close()
	rows, err = db.Query("select xid from filemeta")
	for rows.Next() {
		var xid string
		err = rows.Scan(&xid)
		if err != nil {
			log.Fatal(err)
		}
		delete(filexids, xid)
	}
	rows.Close()
	tx, err := blobdb.Begin()
	if err != nil {
		log.Fatal(err)
	}
	for xid, _ := range filexids {
		_, err = tx.Exec("delete from filedata where xid = ?", xid)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

var stmtHonkers, stmtDubbers, stmtSaveHonker, stmtUpdateFlavor, stmtUpdateCombos *sql.Stmt
var stmtOneXonk, stmtPublicHonks, stmtUserHonks, stmtHonksByCombo, stmtHonksByConvoy *sql.Stmt
var stmtHonksByOntology, stmtHonksForUser, stmtHonksForMe, stmtSaveDub, stmtHonksByXonker *sql.Stmt
var stmtHonksBySearch, stmtHonksByHonker, stmtSaveHonk, stmtWhatAbout *sql.Stmt
var stmtOneBonk, stmtFindZonk, stmtFindXonk, stmtSaveDonk *sql.Stmt
var stmtFindFile, stmtGetFileData, stmtSaveFileData, stmtSaveFile *sql.Stmt
var stmtAddDoover, stmtGetDoovers, stmtLoadDoover, stmtZapDoover, stmtOneHonker *sql.Stmt
var stmtThumbBiters, stmtDeleteHonk, stmtDeleteDonks, stmtDeleteOnts, stmtSaveZonker *sql.Stmt
var stmtGetZonkers, stmtRecentHonkers, stmtGetXonker, stmtSaveXonker, stmtDeleteXonker *sql.Stmt
var stmtSelectOnts, stmtSaveOnt, stmtUpdateFlags, stmtClearFlags *sql.Stmt
var stmtHonksForUserFirstClass, stmtSaveMeta, stmtDeleteMeta, stmtUpdateHonk *sql.Stmt

func preparetodie(db *sql.DB, s string) *sql.Stmt {
	stmt, err := db.Prepare(s)
	if err != nil {
		log.Fatalf("error %s: %s", err, s)
	}
	return stmt
}

func prepareStatements(db *sql.DB) {
	stmtHonkers = preparetodie(db, "select honkerid, userid, name, xid, flavor, combos from honkers where userid = ? and (flavor = 'sub' or flavor = 'peep' or flavor = 'unsub') order by name")
	stmtSaveHonker = preparetodie(db, "insert into honkers (userid, name, xid, flavor, combos) values (?, ?, ?, ?, ?)")
	stmtUpdateFlavor = preparetodie(db, "update honkers set flavor = ? where userid = ? and xid = ? and flavor = ?")
	stmtUpdateCombos = preparetodie(db, "update honkers set combos = ? where honkerid = ? and userid = ?")
	stmtOneHonker = preparetodie(db, "select xid from honkers where name = ? and userid = ?")
	stmtDubbers = preparetodie(db, "select honkerid, userid, name, xid, flavor from honkers where userid = ? and flavor = 'dub'")

	selecthonks := "select honks.honkid, honks.userid, username, what, honker, oonker, honks.xid, rid, dt, url, audience, noise, precis, format, convoy, whofore, flags from honks join users on honks.userid = users.userid "
	limit := " order by honks.honkid desc limit 250"
	butnotthose := " and convoy not in (select name from zonkers where userid = ? and wherefore = 'zonvoy' order by zonkerid desc limit 100)"
	stmtOneXonk = preparetodie(db, selecthonks+"where honks.userid = ? and xid = ?")
	stmtOneBonk = preparetodie(db, selecthonks+"where honks.userid = ? and xid = ? and what = 'bonk' and whofore = 2")
	stmtPublicHonks = preparetodie(db, selecthonks+"where whofore = 2 and dt > ?"+limit)
	stmtUserHonks = preparetodie(db, selecthonks+"where (whofore = 2 or whofore = ?) and username = ? and dt > ?"+limit)
	stmtHonksForUser = preparetodie(db, selecthonks+"where honks.userid = ? and dt > ? and honker in (select xid from honkers where userid = ? and flavor = 'sub' and combos not like '% - %')"+butnotthose+limit)
	stmtHonksForUserFirstClass = preparetodie(db, selecthonks+"where honks.userid = ? and dt > ? and (what <> 'tonk') and honker in (select xid from honkers where userid = ? and flavor = 'sub' and combos not like '% - %')"+butnotthose+limit)
	stmtHonksForMe = preparetodie(db, selecthonks+"where honks.userid = ? and dt > ? and whofore = 1"+butnotthose+limit)
	stmtHonksByHonker = preparetodie(db, selecthonks+"join honkers on (honkers.xid = honks.honker or honkers.xid = honks.oonker) where honks.userid = ? and honkers.name = ?"+butnotthose+limit)
	stmtHonksByXonker = preparetodie(db, selecthonks+" where honks.userid = ? and (honker = ? or oonker = ?)"+butnotthose+limit)
	stmtHonksByCombo = preparetodie(db, selecthonks+"join honkers on honkers.xid = honks.honker where honks.userid = ? and honkers.combos like ?"+butnotthose+limit)
	stmtHonksBySearch = preparetodie(db, selecthonks+"where honks.userid = ? and noise like ?"+limit)
	stmtHonksByConvoy = preparetodie(db, selecthonks+"where (honks.userid = ? or (? = -1 and whofore = 2)) and convoy = ?"+limit)
	stmtHonksByOntology = preparetodie(db, selecthonks+"join onts on honks.honkid = onts.honkid where onts.ontology = ? and (honks.userid = ? or (? = -1 and honks.whofore = 2))"+limit)

	stmtSaveMeta = preparetodie(db, "insert into honkmeta (honkid, genus, json) values (?, ?, ?)")
	stmtDeleteMeta = preparetodie(db, "delete from honkmeta where honkid = ?")
	stmtSaveHonk = preparetodie(db, "insert into honks (userid, what, honker, xid, rid, dt, url, audience, noise, convoy, whofore, format, precis, oonker, flags) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	stmtDeleteHonk = preparetodie(db, "delete from honks where honkid = ?")
	stmtUpdateHonk = preparetodie(db, "update honks set precis = ?, noise = ?, format = ?, dt = ? where honkid = ?")
	stmtSaveOnt = preparetodie(db, "insert into onts (ontology, honkid) values (?, ?)")
	stmtDeleteOnts = preparetodie(db, "delete from onts where honkid = ?")
	stmtSaveDonk = preparetodie(db, "insert into donks (honkid, fileid) values (?, ?)")
	stmtDeleteDonks = preparetodie(db, "delete from donks where honkid = ?")
	stmtSaveFile = preparetodie(db, "insert into filemeta (xid, name, description, url, media, local) values (?, ?, ?, ?, ?, ?)")
	blobdb := openblobdb()
	stmtSaveFileData = preparetodie(blobdb, "insert into filedata (xid, media, content) values (?, ?, ?)")
	stmtGetFileData = preparetodie(blobdb, "select media, content from filedata where xid = ?")
	stmtFindXonk = preparetodie(db, "select honkid from honks where userid = ? and xid = ?")
	stmtFindFile = preparetodie(db, "select fileid from filemeta where url = ? and local = 1")
	stmtWhatAbout = preparetodie(db, "select userid, username, displayname, about, pubkey, options from users where username = ?")
	stmtSaveDub = preparetodie(db, "insert into honkers (userid, name, xid, flavor) values (?, ?, ?, ?)")
	stmtAddDoover = preparetodie(db, "insert into doovers (dt, tries, username, rcpt, msg) values (?, ?, ?, ?, ?)")
	stmtGetDoovers = preparetodie(db, "select dooverid, dt from doovers")
	stmtLoadDoover = preparetodie(db, "select tries, username, rcpt, msg from doovers where dooverid = ?")
	stmtZapDoover = preparetodie(db, "delete from doovers where dooverid = ?")
	stmtThumbBiters = preparetodie(db, "select userid, name, wherefore from zonkers")
	stmtFindZonk = preparetodie(db, "select zonkerid from zonkers where userid = ? and name = ? and wherefore = 'zonk'")
	stmtGetZonkers = preparetodie(db, "select zonkerid, name, wherefore from zonkers where userid = ? and wherefore <> 'zonk'")
	stmtSaveZonker = preparetodie(db, "insert into zonkers (userid, name, wherefore) values (?, ?, ?)")
	stmtGetXonker = preparetodie(db, "select info from xonkers where name = ? and flavor = ?")
	stmtSaveXonker = preparetodie(db, "insert into xonkers (name, info, flavor) values (?, ?, ?)")
	stmtDeleteXonker = preparetodie(db, "delete from xonkers where name = ? and flavor = ?")
	stmtRecentHonkers = preparetodie(db, "select distinct(honker) from honks where userid = ? and honker not in (select xid from honkers where userid = ? and flavor = 'sub') order by honkid desc limit 100")
	stmtUpdateFlags = preparetodie(db, "update honks set flags = flags | ? where honkid = ?")
	stmtClearFlags = preparetodie(db, "update honks set flags = flags & ~ ? where honkid = ?")
	stmtSelectOnts = preparetodie(db, "select distinct(ontology) from onts join honks on onts.honkid = honks.honkid where (honks.userid = ? or honks.whofore = 2)")
}
