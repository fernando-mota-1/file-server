package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/natefinch/lumberjack"
)

var itsLogger *log.Logger
var itsStorageLocation string

func handleRequests() {
	http.Handle("/", http.FileServer(http.Dir("./storage")))
	http.Handle("/upload", endPointWrapper(uploadHandler))
	http.Handle("/receive", endPointWrapper(receiveHandler))
	itsLogger.Fatal(http.ListenAndServe(":9092", nil))
}

func endPointWrapper(endpoint func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		itsLogger.Println("\"" + r.Method + "\" request from \"" + r.RemoteAddr + "\" for path \"" + r.RequestURI + "\"")
		endpoint(w, r)
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "./static/upload.html")
	default:
		msg := fmt.Sprint(r.Method, " method has not yet been implemented")
		itsLogger.Println(msg)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		file, header, err := r.FormFile("filebrowser")
		if err != nil {
			msg := "Error receiving data: " + err.Error()
			fmt.Fprintln(w, msg)
			itsLogger.Println(msg)
			return
		}
		targetDir := r.FormValue("target_loc")
		defer file.Close()
		parentFolder := filepath.Join(itsStorageLocation, targetDir)
		err = os.MkdirAll(parentFolder, 0777)
		if err != nil {
			msg := "Error creating parent directory \"" + targetDir + "\": " + err.Error()
			fmt.Fprintln(w, msg)
			itsLogger.Println(msg)
			return
		}
		out, err := os.Create(filepath.Join(parentFolder, header.Filename))
		if err != nil {
			msg := "Error creating file \"" + header.Filename + "\": " + err.Error()
			fmt.Fprintln(w, msg)
			itsLogger.Println(msg)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			msg := "Error writing to file \"" + header.Filename + "\": " + err.Error()
			fmt.Fprintln(w, msg)
			itsLogger.Println(msg)
			return
		}
		fmt.Fprintln(w, "<html><pre>File uploaded successfully: <a href=\"/"+targetDir+"\">Save Location</a></pre></html>")
	default:
		msg := fmt.Sprint(r.Method, " method has not yet been implemented")
		itsLogger.Println(msg)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}
}

func initLogger(theLogDir string, theLogFileName string, theMaxFileSize int, theMaxBackups int, theMaxAge int) (*log.Logger, error) {
	err := os.MkdirAll(theLogDir, 0777)
	if err != nil {
		fmt.Println("Error creating log directory:", err)
		return nil, err
	}
	aLogFile := filepath.Join(theLogDir, theLogFileName)
	f, err := os.OpenFile(aLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return nil, err
	}
	aLumberJack := &lumberjack.Logger{
		Filename:   aLogFile,
		MaxSize:    theMaxFileSize,
		MaxBackups: theMaxBackups,
		MaxAge:     theMaxAge,
	}
	aLogger := log.New(f, "", log.Ldate|log.Lmicroseconds)
	aLogger.SetOutput(aLumberJack)
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP)

	go func() {
		for {
			<-c
			aLumberJack.Rotate()
		}
	}()

	return aLogger, nil
}

func main() {
	var err error
	itsStorageLocation = "./storage"
	itsLogger, err = initLogger("./logs", "fileserver.log", 5, 3, 28)
	if err != nil {
		errMsg := fmt.Sprint("Error:", err)
		fmt.Println(errMsg)
		os.Exit(1)
	}

	handleRequests()
}
