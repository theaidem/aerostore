aerostore
=========

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions).

## Requirements

Depends on the [aerospike-client-go](https://github.com/aerospike/aerospike-client-go) Aerospike library.

## Installation

	go get github.com/theaidem/aerostore

## Documentation

Available on [godoc.org](http://godoc.org/github.com/theaidem/aerostore).

Also you can see [full documentation](http://www.gorillatoolkit.org/pkg/sessions) on underlying interface.

### Examples

	// Fetch new store.
	// "test" is a namespace of Aerospike cluster
	// "sessions" is a set of Aerospike cluster
	store, err := NewAeroStore("test", "sessions", "127.0.0.1", 3000, []byte("something-very-secret"))
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Get a session.
	session, err = store.Get(req, "session-key")
	if err != nil {
		log.Error(err.Error())
	}

	// Add a value.
	session.Values["foo"] = "bar"

	// Save.
	if err = sessions.Save(req, rsp); err != nil {
		log.Fatalf("Error saving session: %v", err)
	}
