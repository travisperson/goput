package main

import "github.com/codegangsta/martini"

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/hex"
	"crypto/sha256"
	"encoding/json"
	"container/list"
	"time"
)

type FileStore struct {
	file_by_hash	map[string] *File
	hash_by_key		map[string] string
	file_fifo		[]*File
	
	memory_used		int
	memory_limit	int
}

func (*fs FileStore) MakeFile(content []byte, content_type, file_name string) *File {
	file := new (File)

	file.content		= content;
	file.CreatedAt		= time.Now();
	file.ContentType	= content_type
	file.Name			= file_name
	file.Length			= len(file.content)
	file.Hash			= Hash(content);

	MemoryUsed += file.Length

	// Catch all
	if file.ContentType == "application/x-www-form-urlencoded" || len(file.ContentType) == 0 {
		file.ContentType = "application/octet-stream"
	}

	return file
}

func (*fs FileStore) PutFileByHash(file *File) bool {
	append(file_fifo,file)
	return file_by_hash[file.Hash] = file
}

func (*fs FileStore) LinkFileToKey(key string, hash string) bool {
	hash_by_key[key] = hash
}

func (*fs FileStore) GetFileByHash(hash string) *File, bool {
	return fs.file_by_hash[params["id"]]
}

func (*fs FileStore) GetFileByKey(key string) *File, bool {
	hash, ok := fs.GetHashByKey(key)

	if !ok {
		return nil, ok
	}

	return fs.GetFileByHash(hash)
}

func (*fs FileStore) MakeRoomFor(size int) {
	for  fs.memory_used + size > memory_limit {
		f := file_fifo[0];

		delete(file_by_hash, f.Hash)
		delete(file_by_key, f.Hash)
	}
}

func (*fs FileStore) GetHashByKey(key string) string, bool {
	return fs.hash_by_key[key]
}

type File struct {
	ContentType string
	Name string
	CreatedAt time.Time
	content []byte
	Length int
	Hash string
	Key string
}

func Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

type JM map[string] string

func Stringify (data JM) string {
	b,_ := json.Marshal(data)
	return string(b)
}

const (
	B  = (1)
	KB = (1024)
	MB = (1024 * 1024)
	GB = (1024 * 1024 * 1024)
)

func main() {

	fs = new FileStore
	
	m := martini.Classic()

	m.Get("/", func() string {
		return Stringify(JM{"Message": "Hello World"})
	})

	m.Get("/:id", func(res http.ResponseWriter, params martini.Params) (int, []byte) {
		file, ok := Files[params["id"]]

		if !ok {
			return 404, []byte(Stringify(JM{"Message": "The file you are trying to access does not exists."}))
		}

		res.Header().Set("Content-Type", file.ContentType)
		res.Header().Set("Content-Disposition", "Attachment;filename=" + file.Name )

		return 200, file.content
	})

	m.Put("/:id", func(params martini.Params, request *http.Request) (int, []byte) {

		_, ok := Files[params["id"]]

		if ok {
			return 404, []byte(Stringify(JM{"Message": "A file already exists at this location."}))
		}

		content, err := ioutil.ReadAll(request.Body);

		if err != nil {
			fmt.Println(err)
		}
		
		file := new (File)

		file.content		= content;
		file.CreatedAt		= time.Now();
		file.ContentType	= request.Header.Get("Content-Type")
		file.Name			= request.Header.Get("File-Name")
		file.Length			= len(file.content)
		file.Hash			= Hash(content);

		MemoryUsed += file.Length

		// Catch all
		if file.ContentType == "application/x-www-form-urlencoded" || len(file.ContentType) == 0 {
			file.ContentType = "application/octet-stream"
		}

		if len(file.Name) == 0 {
			file.Name = params["id"]
		}
	
		Files[params["id"]] = file

		b, err := json.Marshal(file)
		if err != nil {
			fmt.Println(err)
			return 500, make([]byte,0)
		}

		return 200, b

	})

	m.Post("/", func(request *http.Request) (int, []byte) {
		content, err := ioutil.ReadAll(request.Body);

		if err != nil {
			fmt.Println(err)
		}
		
		file := new (File)

		file.content		= content;
		file.CreatedAt		= time.Now();
		file.ContentType	= request.Header.Get("Content-Type")
		file.Name			= request.Header.Get("File-Name")
		file.Length			= len(file.content)
		file.Hash			= Hash(content);

		MemoryUsed += file.Length

		// Catch all
		if file.ContentType == "application/x-www-form-urlencoded" || len(file.ContentType) == 0 {
			file.ContentType = "application/octet-stream"
		}

		if len(file.Name) == 0 {
			file.Name = file.Hash[0:15]
		}
	
		Files[file.Hash] = file

		b, err := json.Marshal(file)
		if err != nil {
			fmt.Println(err)
			return 500, make([]byte,0)
		}

		return 200, b
	})

  m.Run()
}
