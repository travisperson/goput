package main

import "github.com/codegangsta/martini"

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/hex"
	"crypto/sha256"
	"encoding/json"
	//"container/list"
	"mime"
	"time"
)

type FileStore struct {
	file_by_hash	map[string] *File
	hash_by_key		map[string] string
	file_fifo		[]*File
	
	memory_used		int
	memory_limit	int
}

func (fs *FileStore) MakeFile(content []byte, content_type, file_name string) *File {
	file := new (File)

	file.content		= content
	file.CreatedAt		= time.Now()
	file.ContentType	= content_type
	file.Name			= file_name
	file.Length			= len(file.content)
	file.Hash			= Hash(content)

	fs.memory_used += file.Length

	fmt.Println(content_type)

	
	// Catch all
	if file.ContentType == "application/x-www-form-urlencoded" || len(file.ContentType) == 0 {
		file.ContentType = "application/octet-stream"
	}


	return file
}

func (fs *FileStore) PutFileByHash(file *File) bool {
	fs.file_fifo = append(fs.file_fifo, file)
	fs.file_by_hash[file.Hash] = file
	_, ok := fs.file_by_hash[file.Hash]
	return ok;
}

func (fs *FileStore) LinkFileToKey(key string, hash string) bool {

	f, ok := fs.file_by_hash[hash]

	if !ok {
		return false
	}

	f.Key = key

	fs.hash_by_key[key] = hash
	hash, ok  = fs.hash_by_key[key]
	return ok;
}

func (fs *FileStore) GetFileByHash(hash string) (*File, bool) {
	f, ok := fs.file_by_hash[hash]
	return f, ok
}

func (fs *FileStore) GetFileByKey(key string) (*File, bool) {
	hash, ok := fs.GetHashByKey(key)

	if !ok {
		return nil, ok
	}

	return fs.GetFileByHash(hash)
}

func (fs *FileStore) MakeRoomFor(size int) {
	for  fs.memory_used + size > fs.memory_limit {
		f := fs.file_fifo[0];

		delete(fs.file_by_hash, f.Hash)
		delete(fs.hash_by_key, f.Key)
	}
}

func (fs *FileStore) GetHashByKey(key string) (string, bool) {
	hash, ok := fs.hash_by_key[key]
	return hash, ok
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

	fs := new (FileStore)
	
	fs.file_by_hash	= make(map[string] *File)
	fs.hash_by_key	= make(map[string] string)
	fs.file_fifo	= make([]*File, 128)
	
	m := martini.Classic()

	m.Get("/", func() string {
		return Stringify(JM{"Message": "Hello World"})
	})

	m.Get("/:id", func(res http.ResponseWriter, params martini.Params) (int, []byte) {

		file, ok := fs.GetFileByHash(params["id"])

		if !ok {
			file, ok = fs.GetFileByKey(params["id"])
			if !ok {
				return 404, []byte(Stringify(JM{"Message": "The file you are trying to access does not exists."}))
			}
		}

		res.Header().Set("Content-Type", file.ContentType)
		res.Header().Set("Content-Disposition", "Attachment;filename=" + file.Name )

		return 200, file.content
	})

	m.Put("/:id", func(params martini.Params, request *http.Request) (int, []byte) {
		_, ok := fs.GetFileByKey(params["id"])

		if ok {
			return 404, []byte(Stringify(JM{"Message": "A file already exists at this location."}))
		}

		content, err := ioutil.ReadAll(request.Body);

		if err != nil {
			fmt.Println(err)
		}
		
		file := fs.MakeFile(content, request.Header.Get("Content-Type"), request.Header.Get("File-Name"))

		if len(file.Name) == 0 {
			file.Name = params["id"]
		}
	
		fs.PutFileByHash(file)
		fs.LinkFileToKey(params["id"], file.Hash)

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
		
		file := fs.MakeFile(content, request.Header.Get("Content-Type"), request.Header.Get("File-Name"))

		if len(file.Name) == 0 {
			file.Name = file.Hash[0:15]
		}
	
		fs.PutFileByHash(file)

		b, err := json.Marshal(file)
		if err != nil {
			fmt.Println(err)
			return 500, make([]byte,0)
		}

		return 200, b
	})

  m.Run()
}
