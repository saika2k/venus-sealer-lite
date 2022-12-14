package main

import (
	"encoding/json"
	"github.com/filecoin-project/venus-sealer/api"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

const metaFile = "sectorstore.json"

var storageCmd = &cli.Command{
	Name:  "storage",
	Usage: "manage sector storage",
	Subcommands: []*cli.Command{
		storageAttachCmd,
	},
}

var storageAttachCmd = &cli.Command{
	Name:  "attach",
	Usage: "attach local storage path",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "init",
			Usage: "initialize the path first",
		},
		&cli.Uint64Flag{
			Name:  "weight",
			Usage: "(for init) path weight",
			Value: 10,
		},
		&cli.BoolFlag{
			Name:  "seal",
			Usage: "(for init) use path for sealing",
		},
		&cli.BoolFlag{
			Name:  "store",
			Usage: "(for init) use path for long-term storage",
		},
		&cli.StringSliceFlag{
			Name:  "groups",
			Usage: "path group names",
		},
		&cli.StringSliceFlag{
			Name:  "allow-to",
			Usage: "path groups allowed to pull data from this path (allow all if not specified)",
		},
	},
	Action: func(cctx *cli.Context) error {
		workerApi, closer, err := api.GetWorkerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		if !cctx.Args().Present() {
			return xerrors.Errorf("must specify storage path to attach")
		}

		p, err := homedir.Expand(cctx.Args().First())
		if err != nil {
			return xerrors.Errorf("expanding path: %w", err)
		}

		if cctx.Bool("init") {
			if err := os.MkdirAll(p, 0755); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}

			_, err := os.Stat(filepath.Join(p, metaFile))
			if !os.IsNotExist(err) {
				if err == nil {
					return xerrors.Errorf("path is already initialized")
				}
				return err
			}

			cfg := &stores.LocalStorageMeta{
				ID:       stores.ID(uuid.New().String()),
				Weight:   cctx.Uint64("weight"),
				CanSeal:  cctx.Bool("seal"),
				CanStore: cctx.Bool("store"),
				Groups:   cctx.StringSlice("groups"),
				AllowTo:  cctx.StringSlice("allow-to"),
			}

			if !(cfg.CanStore || cfg.CanSeal) {
				return xerrors.Errorf("must specify at least one of --store or --seal")
			}

			b, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return xerrors.Errorf("marshaling storage config: %w", err)
			}

			if err := ioutil.WriteFile(filepath.Join(p, metaFile), b, 0644); err != nil {
				return xerrors.Errorf("persisting storage metadata (%s): %w", filepath.Join(p, metaFile), err)
			}
		}

		return workerApi.StorageAddLocal(ctx, p)
	},
}
