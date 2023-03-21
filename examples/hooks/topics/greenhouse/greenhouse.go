// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-co
// SPDX-FileContributor: mochi-co

package main

import (
	"bytes"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mochi-co/mqtt/v2"
	"github.com/mochi-co/mqtt/v2/hooks/auth"
	"github.com/mochi-co/mqtt/v2/listeners"
	"github.com/mochi-co/mqtt/v2/packets"
)

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	server := mqtt.New(nil)
	//_ = server.AddHook(new(auth.AllowHook), nil)
  //_ = server.AddHook(new(auth.Hook), nil)
  _ = server.AddHook(new(auth.Hook), &auth.Options{
    Ledger: &auth.Ledger{
    Auth: auth.AuthRules{ // Auth disallows all by default
      {Username: "peach", Password: "password1", Allow: true},
      {Username: "melon", Password: "password2", Allow: true},
      {Remote: "127.0.0.1:*", Allow: true},
      {Remote: "localhost:*", Allow: true},
    },
    ACL: auth.ACLRules{ // ACL allows all by default
      {Remote: "127.0.0.1:*"}, // local superuser allow all
      {
        // user melon can read and write to their own topic
        Username: "melon", Filters: auth.Filters{
          "melon/#":   auth.ReadWrite,
          "updates/#": auth.ReadWrite, // can write to updates, but can't read updates from others
        },
      },
      {
        // Otherwise, no clients have publishing permissions
        Filters: auth.Filters{
          "#":         auth.ReadWrite,
          "updates/#": auth.Deny,
        },
      },
    },
  },
})
	tcp := listeners.NewTCP("t1", ":1883", nil)
	err := server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	err = server.AddHook(new(ExampleHook), map[string]any{})
  
	if err != nil {
		log.Fatal(err)
	}

	// Start the server
	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Demonstration of directly publishing messages to a topic via the
	// `server.Publish` method. Subscribe to `direct/publish/greenhouse` using your
	// MQTT client to see the messages.

//	go func() {
	//	cl := server.NewClient(nil, "local", "inline", true)
	//	for range time.Tick(time.Second * 1) {
		//	err := server.InjectPacket(cl, packets.Packet{
		//		FixedHeader: packets.FixedHeader{
			//		Type: packets.Publish,
			//	},
		//		TopicName: "direct/publish/greenhouse",
		//		Payload:   []byte("injected scheduled message"),
		//	})
		//	if err != nil {
			//	server.Log.Error().Err(err).Msg("server.InjectPacket")
		//	}
		//	server.Log.Info().Msgf("main.go injected packet to direct/publish/greenhouse")
	//	}
//	}()


	// There is also a shorthand convenience function, Publish, for easily sending
	// publish packets if you are not concerned with creating your own packets.
	//go func() {
	//	for range time.Tick(time.Second * 5) {
		//	err := server.Publish("direct/publish/greenhouse", []byte("packet scheduled message"), false, 0)
			//if err != nil {
			//	server.Log.Error().Err(err).Msg("server.Publish")
		//	}
		//	server.Log.Info().Msgf("main.go issued direct message to direct/publish/greenhouse")
	//	}
//	}()


	<-done
	server.Log.Warn().Msg("caught signal, stopping...")
	server.Close()
	server.Log.Info().Msg("main.go finished")
}

type ExampleHook struct {
	mqtt.HookBase
}

func (h *ExampleHook) ID() string {
	return "events-example"
}

func (h *ExampleHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
		mqtt.OnPublish,
	}, []byte{b})
}

func (h *ExampleHook) Init(config any) error {
	h.Log.Info().Msg("initialised")
	return nil
}

func (h *ExampleHook) OnConnect(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Msgf("client connected")
}

func (h *ExampleHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	h.Log.Info().Str("client", cl.ID).Bool("expire", expire).Err(err).Msg("client disconnected")
}

func (h *ExampleHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msgf("subscribed qos=%v", reasonCodes)
}

func (h *ExampleHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msg("unsubscribed")
}

func (h *ExampleHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Str("Topic", string(pk.TopicName)).Msg("received from client")

	pkx := pk
	if string(pk.Payload) == "hello" {
		pkx.Payload = []byte("hello world")
		h.Log.Info().Str("client", cl.ID).Str("payload", string(pkx.Payload)).Msg("received modified packet from client")
	}

	return pkx, nil
}

func (h *ExampleHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Str("Topic", string(pk.TopicName)).Msg("published to client")
}