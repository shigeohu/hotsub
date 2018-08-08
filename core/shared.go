package core

import (
	"context"
	"fmt"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"

	"github.com/otiai10/daap"
	"github.com/otiai10/dkmachine"
	"golang.org/x/sync/errgroup"
)

// SharedData ...
type SharedData struct {
	Spec *dkmachine.CreateOptions

	// {{{ TODO: multiple SharedDataInstances design
	Instance *dkmachine.Machine
	// }}}

	Inputs    Inputs
	Root      string
	Container struct {
		Routine   *daap.Container
		NFSServer *daap.Container
	}
}

// Create ...
func (sd *SharedData) Create() error {

	// FIXME: Use component.Log to manage log level
	fmt.Printf("[Root Component]\t[CREATE]\tCreating Shared Data Instance...\n")

	instance, err := dkmachine.Create(sd.Spec)
	sd.Instance = instance
	if err != nil {
		return err
	}

	eg := new(errgroup.Group)
	eg.Go(sd.startNFS)
	eg.Go(sd.fetchAll)

	return eg.Wait()
}

func (sd *SharedData) fetchAll() error {

	ctx := context.Background()
	container := daap.NewContainer("hotsub/routine", sd.Instance)

	progress, err := container.PullImage(ctx)
	if err != nil {
		return nil
	}

	for range progress {
		// fmt.Printf(".")
	}
	// fmt.Printf("\n")

	err = container.Create(ctx, daap.CreateConfig{
		Host: &dockercontainer.HostConfig{
			Mounts: []mount.Mount{daap.Bind(HOTSUB_MOUNTPOINT, HOTSUB_MOUNTPOINT)},
		},
	})
	if err != nil {
		return err
	}

	if err := container.Start(ctx); err != nil {
		return err
	}

	sd.Container.Routine = container

	eg := new(errgroup.Group)

	for _, input := range sd.Inputs {
		i := input
		eg.Go(func() error { return sd.fetch(i) })
	}

	return eg.Wait()
}

func (sd SharedData) fetch(input *Input) error {

	ctx := context.Background()

	if err := input.Localize(HOTSUB_CONTAINERROOT); err != nil {
		return err
	}

	fetch := &daap.Execution{
		Inline:  "/scripts/download.sh",
		Env:     input.EnvForFetch(),
		Inspect: true,
	}

	stream, err := sd.Container.Routine.Exec(ctx, fetch)
	if err != nil {
		return err
	}
	for payload := range stream {
		fmt.Printf("[SaredDataInstance]\t[FETCH]\t&%d> %s\n", payload.Type, payload.Text())
	}

	if fetch.ExitCode != 0 {
		return fmt.Errorf("fetch in SharedDataInstance exit with %d: %s", fetch.ExitCode, input.URL)
	}

	return nil
}

func (sd SharedData) startNFS() error {

	ctx := context.Background()
	container := daap.NewContainer("otiai10/nfs-server", sd.Instance)

	progress, err := container.PullImage(ctx)
	if err != nil {
		return nil
	}

	for range progress {
		// fmt.Printf(".")
	}
	// fmt.Printf("\n")

	err = container.Create(ctx, daap.CreateConfig{
		Host: &dockercontainer.HostConfig{
			Mounts:      []mount.Mount{daap.Bind(HOTSUB_MOUNTPOINT, HOTSUB_MOUNTPOINT)},
			Privileged:  true,
			NetworkMode: "host",
		},
		Container: &dockercontainer.Config{
			Env: []string{fmt.Sprintf("%s=%s", "MOUNTPOINT", HOTSUB_MOUNTPOINT)},
		},
	})
	if err != nil {
		return err
	}

	if err := container.Start(ctx); err != nil {
		return err
	}

	return nil
}

// CreateNFSVolumesOn creates volumes from `*SharedData` for specified computing machine.
// **This must not mutate `*SharedData` struct itself.**
func (sd *SharedData) CreateNFSVolumesOn(m *dkmachine.Machine) ([]*daap.Volume, error) {
	volumes := []*daap.Volume{}

	volume := &daap.Volume{
		Config: volume.VolumeCreateBody{
			Driver: "local",
			DriverOpts: map[string]string{
				"type":   "nfs",
				"o":      "addr=" + sd.Instance.GetPrivateIPAddress() + ",ro,vers=4",
				"device": ":/",
			},
			Name: "shared",
		},
		Machine: m,
	}

	ctx := context.Background()
	if err := volume.Create(ctx); err != nil {
		return volumes, err
	}

	volumes = append(volumes, volume)
	return volumes, nil
}

// Envs ...
func (sd *SharedData) Envs() (envs []Env) {
	for _, input := range sd.Inputs {
		// Relocalize for workflow container
		input.Localize(HOTSUB_CONTAINERROOT + "/" + HOTSUB_SHARED_DIR)
		envs = append(envs, input.Env())
	}
	return envs
}