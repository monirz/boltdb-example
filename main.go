package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

var db *bolt.DB
var bckt = []byte("bckt")

func init() {
	var err error
	db, err = bolt.Open("my.db", 0600, nil)
	if err != nil {

		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	http.HandleFunc("/values", handler)

	fmt.Println("Listening om :8077")
	http.ListenAndServe(":8077", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		saveValue(w, r)
	} else if r.Method == "GET" {
		getValue(w, r)
	}
}

type jsonValue struct {
	Values string
}

func saveValue(w http.ResponseWriter, r *http.Request) {

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bckt)
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(r.Body)

		var jsonData map[string]string

		err = decoder.Decode(&jsonData)

		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return err
		}

		fmt.Println(jsonData)

		for k, v := range jsonData {

			key := []byte(k)
			value := []byte(v)

			err = bucket.Put(key, value)
			if err != nil {
				return err
			}
		}

		var timeout = time.NewTimer(5 * time.Minute).C

		go func() {

			<-timeout

			deleteValue(jsonData)

		}()
		return nil
	})

	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	str := `{"success": "value added successfully"}`
	w.Header().Set("Content-Type", "aplication/json")
	w.WriteHeader(201)
	w.Write([]byte(str))

}

func getValue(w http.ResponseWriter, r *http.Request) {

	// retrieve the data

	queryParamKey := r.URL.Query().Get("keys")

	if len(queryParamKey) > 0 {

		getValueByKey(w, r, queryParamKey)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bckt)
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found!", bckt)
		}

		c := bucket.Cursor()

		// value := bucket.Get(key)
		values := make(map[string]string)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if string(v) != "" {

				values[string(k)] = string(v)
			}
		}

		if len(values) < 1 {
			w.WriteHeader(404)
			return errors.New("no value found")
		}

		b, err := json.Marshal(values)

		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(500)
			return err
		}
		w.WriteHeader(200)
		w.Write([]byte(b))
		return nil
	})

	if err != nil {
		if err != nil {
			str := `{"error": ` + err.Error() + `}`
			w.Write([]byte(str))
		}
	}

}

func deleteValue(jsonValue map[string]string) {
	fmt.Println("executing delete values")
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bckt))

		for k := range jsonValue {
			err := b.Delete([]byte(k))
			if err != nil {
				return err
			}
		}
		return nil
	})

}

func getValueByKey(w http.ResponseWriter, r *http.Request, queryParams string) {

	w.Header().Set("Content-Type", "application/json")

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bckt)
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found!", bckt)
		}

		// value := bucket.Get(key)

		values := make(map[string]string)

		params := strings.Split(queryParams, ",")

		for _, v := range params {
			val := bucket.Get([]byte(v))
			fmt.Println(string(val))
			if string(val) != "" {

				values[string(v)] = string(val)
			}

		}

		if len(values) < 1 {
			w.WriteHeader(404)
			return errors.New("value not found with the provided keys")
		}

		b, err := json.Marshal(values)

		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(500)
			return err
		}
		w.WriteHeader(200)
		w.Write([]byte(b))
		return nil
	})

	if err != nil {
		log.Println(err)
		str := `{"error": ` + err.Error() + `}`
		w.Write([]byte(str))
	}

}
