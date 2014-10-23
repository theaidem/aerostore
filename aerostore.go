// Copyright 2012 Max "theaidem" Kokorin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aerostore

import (
	"bytes"
	"encoding/base32"
	"encoding/gob"
	"errors"
	"net/http"
	"strings"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var notConnError = errors.New("SessionStore: Aerospike client isn't connected to the cluster")

// AeroStore stores sessions in a Aerospike backend.
type AeroStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options // default configuration
	client  *as.Client
	ns, set string
}

// NewAeroStore returns a new AeroStore.
// ns: Aerospike namespace (similar to database)
// set: Aerospike set (similar to table)
func NewAeroStore(ns, set string, host string, port int, keyPairs ...[]byte) (*AeroStore, error) {
	client, err := as.NewClient(host, port)
	if err != nil {
		return nil, err
	}
	return NewAeroStoreWithClient(ns, set, client, keyPairs...)
}

// NewAeroStoreWithClient instantiates a AeroStore with a *as.Client passed in.
func NewAeroStoreWithClient(ns, set string, client *as.Client, keyPairs ...[]byte) (*AeroStore, error) {
	var err error
	if !client.IsConnected() {
		err = notConnError
	}
	return &AeroStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		client: client,
		ns:     ns,
		set:    set,
	}, err
}

// Close closes the underlying the Aerospike client
func (s *AeroStore) Close() {
	s.client.Close()
}

// Get returns a session for the given name after adding it to the registry.
//
// See CookieStore.Get().
func (s *AeroStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// See CookieStore.New().
func (s *AeroStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var err error
	session := sessions.NewSession(s, name)
	options := *s.Options
	session.Options = &options
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			ok, err := s.load(session)
			session.IsNew = !(err == nil && ok) // not new if no error and data available
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *AeroStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	var err error
	// Build an alphanumeric key for the Aerospike store.
	if session.ID == "" {
		session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	}
	if err = s.save(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))

	return nil
}

// save stores the session in Aerospike.
func (s *AeroStore) save(session *sessions.Session) error {
	if !s.client.IsConnected() {
		return notConnError
	}

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(session.Values)
	if err != nil {
		return err
	}
	b := buf.Bytes()

	key, err := as.NewKey(s.ns, s.set, "session_"+session.ID)
	if err != nil {
		return err
	}

	bin := as.NewBin("data", b)
	err = s.client.PutBins(nil, key, bin)
	if err != nil {
		return err
	}
	return nil
}

// load stores the session in Aerospike.
func (s *AeroStore) load(session *sessions.Session) (bool, error) {
	if !s.client.IsConnected() {
		return false, notConnError
	}

	key, err := as.NewKey(s.ns, s.set, "session_"+session.ID)
	if err != nil {
		return false, err
	}

	rec, err := s.client.Get(nil, key)
	if err != nil {
		return false, err
	}

	if rec != nil {
		dec := gob.NewDecoder(bytes.NewBuffer(rec.Bins["data"].([]byte)))
		return true, dec.Decode(&session.Values)
	}
	return false, nil
}
