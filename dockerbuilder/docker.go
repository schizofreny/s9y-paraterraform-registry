package dockerbuilder

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/thoas/go-funk"
)

// GoDockerContainer struct
type GoDockerContainer struct {
	containerID string
	client      *client.Client
	ctx         context.Context
}

// NewGoDockerContainer - Start new container for modules builds
func NewGoDockerContainer(Goversion string) (*GoDockerContainer, error) {

	cli, err := client.NewEnvClient()
	d := &GoDockerContainer{
		ctx:    context.Background(),
		client: cli,
	}

	if err != nil {
		return nil, err
	}

	err = d.init(Goversion)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *GoDockerContainer) init(goVersion string) error {
	image := "docker.io/library/golang"
	if goVersion != "" {
		image = fmt.Sprintf("%s:%s", image, goVersion)
	}

	reader, err := d.client.ImagePull(d.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, reader)

	log.Println("----> Creating container")
	resp, err := d.client.ContainerCreate(d.ctx, &container.Config{
		Image: image,
		Cmd:   []string{"sleep", "500"},
		Tty:   true,
	}, nil, nil, "")

	if err != nil {
		return err
	}

	d.containerID = resp.ID

	log.Println("----> Starting container")

	err = d.client.ContainerStart(d.ctx, d.containerID, types.ContainerStartOptions{})
	return err

}

// Kill kills current container
func (d *GoDockerContainer) Kill() {

	d.client.ContainerKill(d.ctx, d.containerID, "SIGKILL")
}

// Exec exec command in workdir
func (d *GoDockerContainer) Exec(command []string, workdir string) error {
	log.Printf("=====> %v", command)
	cfg := types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          command,
		WorkingDir:   workdir,
	}
	out, err := d.client.ContainerLogs(d.ctx, d.containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	r, err := d.client.ContainerExecCreate(d.ctx, d.containerID, cfg)
	if err != nil {
		return err

	}

	hrr, err := d.client.ContainerExecAttach(d.ctx, r.ID, types.ExecStartCheck{})
	if err != nil {
		return err

	}

	io.Copy(os.Stdout, hrr.Reader)

	return nil
}

// Convert filet to tar and copy to container
// https: //github.com/docker/cli/blob/b1d27091e50595fecd8a2a4429557b70681395b2/cli/command/container/cp.go#L255
func (d *GoDockerContainer) copyFileToContainer(src string, dst string) error {
	dstInfo := archive.CopyInfo{Path: dst}
	srcInfo, err := archive.CopyInfoSourcePath(src, false)
	if err != nil {
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return err
	}
	defer srcArchive.Close()

	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		return err
	}
	defer preparedArchive.Close()

	resolvedDstPath := dstDir
	content := preparedArchive
	err = d.client.CopyToContainer(d.ctx, d.containerID, resolvedDstPath, content, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
	if err != nil {
		return err
	}
	return nil
}

// ExecShellScriptLines copies shell script to container temp and evaluate it
func (d *GoDockerContainer) ExecShellScriptLines(lines []string, workdir string) error {

	file, err := ioutil.TempFile("", "parateraregsh")
	if err != nil {
		return err
	}
	defer file.Close()
	defer os.Remove(file.Name())

	file.WriteString("#!/bin/bash\nset -e\nset -x\n\n")
	funk.ForEach(lines, func(s string) {
		file.WriteString(s + "\n")
	})

	file.Close()

	if err != nil {
		return err
	}

	err = d.copyFileToContainer(file.Name(), "/tmp/x.sh")
	if err != nil {
		return err
	}

	err = d.Exec([]string{"/bin/bash", "/tmp/x.sh"}, workdir)
	if err != nil {
		return err
	}
	return nil

}

// CopyFromContainer copies file from container
func (d *GoDockerContainer) CopyFromContainer(srcPath string, dstPath string) (err error) {

	// if client requests to follow symbol link, then must decide target file to be copied
	var rebaseName string

	content, _, err := d.client.CopyFromContainer(d.ctx, d.containerID, srcPath)
	if err != nil {
		return err
	}
	defer content.Close()

	srcInfo := archive.CopyInfo{
		Path:       srcPath,
		Exists:     true,
		IsDir:      false,
		RebaseName: rebaseName,
	}

	preArchive := content
	if len(srcInfo.RebaseName) != 0 {
		_, srcBase := archive.SplitPathDirEntry(srcPath)
		preArchive = archive.RebaseArchiveEntries(content, srcBase, srcInfo.RebaseName)
	}
	return archive.CopyTo(preArchive, srcInfo, dstPath)
}

func (d *GoDockerContainer) execScripts(scripts [][]string, workdir string) {
	for _, s := range scripts {
		d.Exec(s, workdir)
	}
}
