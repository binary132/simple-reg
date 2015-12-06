package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/BurntSushi/toml"
	htr "github.com/julienschmidt/httprouter"
	"github.com/mailgun/mailgun-go"
)

var (
	config = flag.String("config", "config.toml", "The config to use")

	domain   = flag.String("domain", "", "The domain to send from")
	mgAPIKey = flag.String("api-key", "", "The API key to use")
	pbAPIKey = flag.String("pub-key", "", "The public API key to use")
	sendTo   = flag.String("send-to", "", "The email address to send to")
	sender   = flag.String("sender", "", "The email address to send from")
	subject  = flag.String("subject", "Registration", "Subject")

	port = flag.Int("port", 24000, "The port to listen on")
)

type Config struct {
	Domain  string `toml:"domain"`
	APIKey  string `toml:"api-key"`
	PubKey  string `toml:"pub-key"`
	SendTo  string `toml:"send-to"`
	Sender  string `toml:"sender"`
	Subject string `toml:"subject"`
	Port    int    `toml:"port"`
}

func main() {
	flag.Parse()

	conf, err := getConfig(*config)
	if err != nil {
		panic(err)
	}

	m := mailer(
		conf.Domain,
		conf.APIKey,
		conf.PubKey,
		conf.Subject,
		conf.SendTo,
		conf.Sender,
	)
	r := htr.New()
	r.POST("/register/:email", wrap(m, "email"))

	log.Printf("Simple Reg listening on port :%d\n", conf.Port)
	panic(http.ListenAndServe(":"+strconv.Itoa(conf.Port), r))
}

func wrap(f func(string) error, key string) htr.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps htr.Params) {
		if err := f(ps.ByName(key)); err != nil {
			http.Error(w, err.Error(), 500)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func getConfig(which string) (*Config, error) {
	var conf Config
	if which != "" {
		if _, err := toml.DecodeFile(which, &conf); err != nil {
			return nil, fmt.Errorf("failed to parse config %s: %v", which, err)
		}
	}
	for k, v := range map[string]struct {
		f  string
		cf *string
	}{
		"domain":  {f: *domain, cf: &(conf.Domain)},
		"api-key": {f: *mgAPIKey, cf: &(conf.APIKey)},
		"pub-key": {f: *pbAPIKey, cf: &(conf.PubKey)},
		"send-to": {f: *sendTo, cf: &(conf.SendTo)},
		"sender":  {f: *sender, cf: &(conf.Sender)},
	} {
		switch {
		case v.f == "" && *(v.cf) == "":
			return nil, fmt.Errorf("must define a value for flag %s", k)
		case *v.cf == "":
			*v.cf = v.f
		}
	}
	if *port != 0 {
		conf.Port = *port
	}

	return &conf, nil
}

func mailer(domain, apiKey, pubKey, sub, to, from string) func(string) error {
	mg := mailgun.NewMailgun(
		domain,
		apiKey,
		pubKey,
	)

	return func(email string) error {
		m := mg.NewMessage(
			// From
			"Registration <"+from+">",
			// Subject
			sub,
			// Plain-text body
			email,
		)

		if err := m.AddRecipient(to); err != nil {
			log.Printf("failed to add recipient %q: %v", to, err)
			return err
		}

		if _, _, err := mg.Send(m); err != nil {
			log.Printf("failed to send to recipient %q: %v", to, err)
			return err
		}

		log.Printf("registered user %q OK", email)

		return nil
	}
}
