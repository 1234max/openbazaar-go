package core

import (
	"time"

	"github.com/op/go-logging"
	libp2p "gx/ipfs/QmTW4SdgBWq9GjsBsHeUx8WuGxzhgzAf88UMH2w62PC8yK/go-libp2p-crypto"
	"gx/ipfs/QmYVXrKrKHDC9FobgmcmshCDyWwdrfwfanNQN4oxJ9Fk3h/go-libp2p-peer"

	"github.com/OpenBazaar/openbazaar-go/net"
	"github.com/OpenBazaar/openbazaar-go/pb"
	"github.com/OpenBazaar/openbazaar-go/repo"
)

const (
	scannerTestingInterval = time.Duration(1) * time.Minute
	scannerRegularInterval = time.Duration(10) * time.Minute
)

type inboundMessageScanner struct {
	// PerformTask dependencies
	datastore repo.Datastore
	service   net.NetworkService
	broadcast chan repo.Notifier

	// Worker-handling dependencies
	intervalDelay time.Duration
	logger        *logging.Logger
	watchdogTimer *time.Ticker
	stopWorker    chan bool
}

// StartInboundMsgScanner - start the notifier
func (n *OpenBazaarNode) StartInboundMsgScanner() {
	n.InboundMsgScanner = &inboundMessageScanner{
		datastore:     n.Datastore,
		service:       n.Service,
		broadcast:     n.Broadcast,
		intervalDelay: n.scannerIntervalDelay(),
		logger:        logging.MustGetLogger("inboundMessageScanner"),
	}
	go n.InboundMsgScanner.Run()
}

func (n *OpenBazaarNode) scannerIntervalDelay() time.Duration {
	if n.TestnetEnable {
		return scannerTestingInterval
	}
	return scannerRegularInterval
}

func (scanner *inboundMessageScanner) Run() {
	scanner.watchdogTimer = time.NewTicker(scanner.intervalDelay)
	scanner.stopWorker = make(chan bool)

	// Run once on start, then wait for watchdog
	scanner.PerformTask()
	for {
		select {
		case <-scanner.watchdogTimer.C:
			scanner.PerformTask()
		case <-scanner.stopWorker:
			scanner.watchdogTimer.Stop()
			return
		}
	}
}

func (scanner *inboundMessageScanner) Stop() {
	scanner.stopWorker <- true
	close(scanner.stopWorker)
}

func (scanner *inboundMessageScanner) PerformTask() {
	msgs, err := scanner.datastore.Messages().GetAllErrored()
	if err != nil {
		scanner.logger.Error(err)
	} else {
		for _, m := range msgs {
			if m.MsgErr.Error() == ErrInsufficientFunds.Error() {

				// Get handler for this msg type
				handler := scanner.service.HandlerForMsgType(pb.Message_MessageType(m.MessageType))
				if handler != nil {
					pubkey, err := libp2p.UnmarshalPublicKey(m.PeerPubkey)
					if err != nil {
						log.Errorf("Error processing message %s. Type %s: %s", m, m.MessageType, err.Error())
						continue
					}
					i, err := peer.IDFromPublicKey(pubkey)
					if err != nil {
						log.Errorf("Error processing message %s. Type %s: %s", m, m.MessageType, err.Error())
						continue
					}
					msg := new(repo.Message)

					if len(m.Message) > 0 {
						err = msg.UnmarshalJSON(m.Message)
						if err != nil {
							log.Errorf("Error processing message %s. Type %s: %s", m, m.MessageType, err.Error())
							continue
						}
					}

					// Dispatch handler
					_, err = handler(i, &msg.Msg, nil)
					if err != nil {
						log.Debugf("%d handle message error from %s: %s", m.MessageType, m.PeerID, err)
					}
					/*
						// If nil response, return it before serializing
						if rpmes == nil {
							continue
						}

						// give back request id
						rpmes.RequestId = msg.Msg.RequestId
						rpmes.IsResponse = true

						ms, err := scanner.service.messageSenderForPeer(scanner.service.ctx, i)
						if err != nil {
							log.Error("Error getting message sender and opening stream to peer")
							continue
						}

						// send out response msg
						log.Debugf("sending response message to: %s", m.PeerID)
						if err := ms.SendMessage(scanner.service.ctx, rpmes); err != nil {
							log.Debugf("send response error: %s", err)
							continue
						}
					*/
				}

			}
		}
	}
}
