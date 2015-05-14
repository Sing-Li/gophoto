package main

import (
    "fmt"
    "io"
    "net/http"
    "io/ioutil"
    "os"
    "log"
    "encoding/json"
    "github.com/codegangsta/martini"
    "github.com/codegangsta/martini-contrib/render"
    "github.com/ncw/swift"
)
const (
    albumfolder = "myphotos"
	DEFAULT_PORT =  "3000"
	DEFAULT_HOST = "0.0.0.0"
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
           return err2;
        }

        _, err := c.ObjectGet("myphotos", filename + ".jpg", res, false, nil );


        if err != nil {
            return err;
        }
        return nil;

}
type ObjStoreInfo struct {
    Credentials ObjStoreCredentials `json:"credentials"`
}
 
type ObjStoreCredentials struct {
    Authuri    string `json:"auth_uri"`
    Username     string `json:"username"`
    Password     string `json:"password"`
    Globaluri string `json:"global_account_auth_uri"`
}


func main() {

    // check for VCAP_SERVICES, parse it if exists

  
   var runport  string
   var runhost string

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


    
     s := os.Getenv("VCAP_SERVICES")

    if (s != "") {
     
     log.Printf("Found VCAP SERVICES, parsing ....\n");

     services := make(map[string][]ObjStoreInfo)
    err := json.Unmarshal([]byte(s), &services)
    if err != nil {
        log.Printf("Error parsing  connection information: %v\n", err.Error())
        panic(err)
    }

    info := services["objectstorage"]
    if len(info) == 0 {
        log.Printf("No objectstorage services are bound to this application.\n")
        return
    }
 
    creds := info[0].Credentials

     authurl = creds.Authuri
     username = creds.Username
     password = creds.Password
     globalurl = creds.Globaluri
    } else {

        log.Printf("No VCAP SERVICES, using defaults.\n")
    }

   log.Printf("Using host %v+\n", runhost)
    log.Printf("Using port %v+\n", runport)
   log.Printf("Using authurl %v+\n", authurl)
   log.Printf("Using username %v+\n", username)
log.Printf("Using password %v+\n", password)

log.Printf("Global URI is  %v+\n", globalurl)

    c := swift.Connection{
                UserName: username,
                ApiKey:   password,
                AuthUrl:  authurl,
        }

        // Authenticate
        err2 := c.Authenticate()
        if err2 != nil {
           panic(err2)
        }


    containers, err := c.ContainersAll(nil)
     if err != nil {
        panic(err)
     }
    if (len(containers) == 0) {
       log.Printf("no container, creating one")
       err = c.ContainerCreate(albumfolder, nil)
       if err != nil {
        panic(err)
       }
    } else {
        log.Printf("container already exists")
    }
  
    m := martini.Classic()

    m.Use(render.Renderer(render.Options{
        Directory: "tmpl", // Specify what path to load the templates from.
        Layout: "layout", // Specify a layout template. Layouts can call {{ yield }} to render the current template.
        Charset: "UTF-8", // Sets encoding for json and html content-types.
    }))

// public and assets are static directories

    m.Get("/", func(r render.Render) {
        fmt.Printf("%v\n", "g./")
        r.HTML(200, "hello", "world")
    })

     m.Get("/upload", func(r render.Render) {
        fmt.Printf("%v\n", "g./upload")
        r.HTML(200, "upload", "" )
    })
    m.Get("/album", func(r render.Render) {
        fmt.Printf("%v\n", "g./album")
        r.HTML(200, "album", "" )
    })
     




   m.Get("/pic/:who.jpg", func(args martini.Params, res http.ResponseWriter, req *http.Request) {
       // res.Header().Set("Content-Type", "image/jpeg")
       
        c := swift.Connection{
                UserName: username,
                ApiKey:   password,
                AuthUrl:  authurl,
        }

        // Authenticate
        err2 := c.Authenticate()
        if err2 != nil {
           panic(err2)
        }

        _, err3 := c.ObjectGet(albumfolder, args["who"] + ".jpg", res, false, nil );


        if err3 != nil {
            res.WriteHeader(500)
        }
    });

    m.Post("/up", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("%v\n", "p./up")

        file, header, err := r.FormFile("fileToUpload")


        defer file.Close()

        if err != nil {
            fmt.Fprintln(w, err)
            return
        }

        out, err := os.Create("/tmp/file")
        if err != nil {
            fmt.Fprintf(w, "Failed to open the file for writing")
            return
        }
        defer out.Close()
        _, err = io.Copy(out, file)
        if err != nil {
            fmt.Fprintln(w, err)
        }

        // the header contains useful info, like the original file name
        fmt.Fprintf(w, "File %s uploaded successfully.", header.Filename)

        // read file into byte array
        bytearray, err := ioutil.ReadFile("/tmp/file")

        if err != nil {
            fmt.Fprintln(w, err)
            return
        }


		c := swift.Connection{
    			UserName: username,
    			ApiKey:   password,
    			AuthUrl:  authurl,
		}

		// Authenticate
		err2 := c.Authenticate()
		if err2 != nil {
    		panic(err2)
		}
		containers, err2 := c.ContainerNames(nil)
		fmt.Fprintf(w, "containers are %s" , containers)

        err3 := c.ObjectPutBytes(albumfolder, header.Filename, bytearray, "image-jpeg");

        if err3 !=nil {
            panic(err3)
        }




    })


        
    m.RunOnAddr(runhost + ":" + runport)
}

