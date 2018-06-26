package ipfs

import (
	"context"
	"github.com/OpenBazaar/go-ipfs/core/corerepo"
	"github.com/ipfs/go-ipfs/core"
)

/* Recursively un-pin a directory given its hash.
   This will allow it to be garbage collected. */
func UnPinDir(n *core.IpfsNode, rootHash string) error {
	_, err := corerepo.Unpin(n, context.Background(), []string{"/ipfs/" + rootHash}, true)
	return err
}
