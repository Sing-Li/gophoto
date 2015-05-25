package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/ncw/swift"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	albumfolder     = "myphotos"
	DEFAULT_PORT    = "3000"
	DEFAULT_HOST    = "0.0.0.0"
	CREDS_ENV       = "V2CREDS"
	UPLOAD_DISABLED = "UPDISABLE"
)

func getpic(res io.Writer, filename string) error {
	c := swift.Connection{
		UserName: "test:tester",
		ApiKey:   "testing",
		AuthUrl:  "http://127.0.0.1:12345/auth/v1.0",
	}

	// Authenticate
	err2 := c.Authenticate()
	if err2 != nil {
		return err2
	}

	_, err := c.ObjectGet("myphotos", filename+".jpg", res, false, nil)

	if err != nil {
		return err
	}
	return nil

}

type ObjStoreV2Info struct {
	Authurl     string                `json:"auth_url"`
	Swifturl    string                `json:"swift_url"`
	Sdkauthurl  string                `json:"sdk_url"`
	Project     string                `json:"project"`
	Region      string                `json:"region"`
	Credentials ObjStoreV2Credentials `json:"credentials"`
}

type ObjStoreV2Credentials struct {
	Userid   string `json:"userid"`
	Password string `json:"password"`
}

type ObjStoreInfo struct {
	Credentials ObjStoreCredentials `json:"credentials"`
}

type ObjStoreCredentials struct {
	Authuri   string `json:"auth_uri"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Project   string `json:"project"`
	Globaluri string `json:"global_account_auth_uri"`
}

func readPhotoToTempfile(file io.Reader) (string, error) {
	out, err := ioutil.TempFile("", "gophoto-") //os.Create("/tmp/file")
	if err != nil {
		log.Printf("Failed to open temp file for writing - %v", err)
		return "", err
	}
	tmpFilename := out.Name()

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		log.Printf("Failed to copy from input to temp file - %v", err)
		return "", err
	}

	return tmpFilename, nil
}

func sendTempfileToArchive(tmpfilename string, archivefilename string, authenticated swift.Connection) error {
	defer os.Remove(tmpfilename)

	// read file into byte array
	bytearray, err := ioutil.ReadFile(tmpfilename)

	if err != nil {
		log.Printf("Failed to read temp file to bytearray - %v", err)
		return err
	}

	err2 := authenticated.ObjectPutBytes(albumfolder, archivefilename, bytearray, "image-jpeg")
	if err2 != nil {
		log.Printf("Failed to send temp file to archive - %v", err2)
		return err2
	}

	return nil

}
func main() {

	// check for VCAP_SERVICES, parse it if exists

	var runport string
	var runhost string
	var project string

	username := "test:tester"
	password := "testing"
	authurl := "http://127.0.0.1:12345/auth/v1.0"
	globalurl := "http://127.0.0.1:12345/auth/v1.0"

	if runport = os.Getenv("VCAP_APP_PORT"); len(runport) == 0 {
		runport = DEFAULT_PORT
	}
	if runhost = os.Getenv("VCAP_APP_HOST"); len(runhost) == 0 {
		runhost = DEFAULT_HOST
	}

	s := os.Getenv(CREDS_ENV)

	u := os.Getenv(UPLOAD_DISABLED)

	if s != "" {

		log.Printf("Found " + CREDS_ENV + " environment variable, parsing ....\n")

		objstorev2 := make(map[string]ObjStoreV2Info)
		err := json.Unmarshal([]byte(s), &objstorev2)
		if err != nil {
			log.Printf("Error parsing  connection information: %v\n", err.Error())
			panic(err)
		}

		info := objstorev2["CloudIntegration"]
		if &info == nil {
			log.Printf("No cloud integration services accessible to this application.\n")
			return
		}

		creds := info.Credentials
		authurl = info.Authurl + "/v2.0"
		username = creds.Userid
		password = creds.Password
		project = info.Project

	} else {
		log.Printf("No " + CREDS_ENV + ", using defaults.\n")
	}

	log.Printf("Using host %v+\n", runhost)
	log.Printf("Using port %v+\n", runport)
	log.Printf("Using authurl %v+\n", authurl)
	log.Printf("Using username %v+\n", username)
	log.Printf("Using password %v+\n", password)
	log.Printf("Global URI is  %v+\n", globalurl)
	m := martini.Classic()

	c := swift.Connection{
		UserName: username,
		ApiKey:   password,
		AuthUrl:  authurl,
		Tenant:   project,
	}

	// Authenticate
	err2 := c.Authenticate()
	if err2 != nil {
		log.Printf("authenticate error: ", err2)
		panic(err2)
	}

	m.Get("/try2", func(res http.ResponseWriter, req *http.Request) {
		clreq, err := http.NewRequest("GET", authurl+"/firstuser", nil)
		clreq.SetBasicAuth(username, password)
		cli := &http.Client{}
		clresp, err := cli.Do(clreq)

		if err != nil {
			log.Printf("Error fetching crendentials: %v\n", err.Error())
			panic(err)
		}

		fmt.Fprintf(res, "completed successfully.<br>%s<br>", clresp)

	})

	m.Get("/create", func(res http.ResponseWriter, req *http.Request) {

		containers, err := c.ContainersAll(nil)
		if err != nil {
			log.Printf("container list error: ", err)
			panic(err)
		}
		if len(containers) == 0 {
			log.Printf("no container, creating one")
			fmt.Fprintf(res, "no container, creating one<br>")
			err = c.ContainerCreate(albumfolder, nil)
			if err != nil {
				log.Printf("contaienr create error: ", err)
				panic(err)
			}
		} else {
			log.Printf("container already exists")
			fmt.Fprintf(res, "container already exists<br>")
		}

		fmt.Fprintf(res, "completed successfully.<br>")

	})

	m.Use(render.Renderer(render.Options{
		Directory: "tmpl",   // Specify what path to load the templates from.
		Layout:    "layout", // Specify a layout template. Layouts can call {{ yield }} to render the current template.
		Charset:   "UTF-8",  // Sets encoding for json and html content-types.
		Funcs: []template.FuncMap{
			{"getPhotoNames": func() template.HTML {

				var result = ""
				var leadstr = "<div class=\"m-item "
				var endstr = "\"><img src=\"pic/"
				var firstcls = "m-active"

				var endtag = "\"></div>"

				files, err := c.ObjectNamesAll("myphotos", nil)
				if err != nil {
					panic(err)
				}

				for i, file := range files {
					result = result + leadstr
					if i == 1 {
						result = result + firstcls
					}
					result = result + endstr + file + endtag

				}
				log.Printf("result is %v", result)
				return template.HTML(result)

			},
			},
		},
	}))

	if u != "" {
		m.Get("/upload", func(r render.Render) {
			fmt.Printf("%v\n", "g./upload")
			r.HTML(200, "uploaddisabled", "")
		})

	} else {
		m.Get("/upload", func(r render.Render) {
			fmt.Printf("%v\n", "g./upload")
			r.HTML(200, "upload", "")
		})
	}

	m.Get("/", func(r render.Render) {
		fmt.Printf("%v\n", "g./album")
		r.HTML(200, "album", "")
	})

	m.Get("/pic/:who.jpg", func(args martini.Params, res http.ResponseWriter, req *http.Request) {
		// res.Header().Set("Content-Type", "image/jpeg")

		_, err3 := c.ObjectGet(albumfolder, args["who"]+".jpg", res, false, nil)

		if err3 != nil {
			res.WriteHeader(500)
		}
	})

	m.Post("/up", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v\n", "p./up")

		file, header, err := r.FormFile("fileToUpload")

		defer file.Close()

		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		tmpFilename, err := readPhotoToTempfile(file)

		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		err2 := sendTempfileToArchive(tmpFilename, header.Filename, c)

		if err2 != nil {
			fmt.Fprintln(w, err2)
			return
		}

		http.Redirect(w, r, "/", 302)

	})

	m.RunOnAddr(runhost + ":" + runport)
}
