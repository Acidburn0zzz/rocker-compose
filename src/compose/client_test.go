package compose

import (
	"compose/config"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cli, err := NewClient(&ClientCfg{})
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &ClientCfg{}, cli)
}

func TestClientGetContainers(t *testing.T) {
	dockerCli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&ClientCfg{Docker: dockerCli, Global: false})
	if err != nil {
		t.Fatal(err)
	}

	containers, err := cli.GetContainers()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []*Container{}, containers)

	// assert.IsType(t, &docker.Env{}, info)
	// fmt.Printf("Containers: %+q\n", containers)
	// pretty.Println(containers)
	// for _, container := range containers {
	// 	data, err := yaml.Marshal(container.Config)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// println(string(data))
	// }

}

func TestClientRunContainer(t *testing.T) {
	t.Skip()

	dockerCli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&ClientCfg{Docker: dockerCli, Global: false})
	if err != nil {
		t.Fatal(err)
	}

	yml := `
namespace: test
containers:
  main:
    image: "busybox:buildroot-2013.08.1"
    labels:
      foo: bar
      xxx: yyy
`

	config, err := config.ReadConfig("test.yml", strings.NewReader(yml), map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range GetContainersFromConfig(config) {
		if err := cli.RunContainer(container); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClientClean(t *testing.T) {
	// This test involves interaction with docker
	// enable it in case you want to test Clean() functionality
	t.Skip()

	dockerCli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	// Create number of images to test

	createdContainers := []string{}
	createdImages := []string{}

	defer func() {
		for _, id := range createdContainers {
			if err := dockerCli.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true}); err != nil {
				t.Error(err)
			}
		}
		for _, id := range createdImages {
			if err := dockerCli.RemoveImageExtended(id, docker.RemoveImageOptions{Force: true}); err != nil {
				if err.Error() == "no such image" {
					continue
				}
				t.Error(err)
			}
		}
	}()

	for i := 1; i <= 5; i++ {
		c, err := dockerCli.CreateContainer(docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: "gliderlabs/alpine:3.1",
				Cmd:   []string{"true"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		createdContainers = append(createdContainers, c.ID)

		commitOpts := docker.CommitContainerOptions{
			Container:  c.ID,
			Repository: "rocker-compose-test-image-clean",
			Tag:        fmt.Sprintf("%d", i),
		}

		img, err := dockerCli.CommitContainer(commitOpts)
		if err != nil {
			t.Fatal(err)
		}

		createdImages = append(createdImages, img.ID)

		// Make sure images have different timestamps
		time.Sleep(time.Second)
	}

	////////////////////////

	cli, err := NewClient(&ClientCfg{Docker: dockerCli, KeepImages: 2})
	if err != nil {
		t.Fatal(err)
	}

	yml := `
namespace: test
containers:
  main:
    image: rocker-compose-test-image-clean:5
`

	config, err := config.ReadConfig("test.yml", strings.NewReader(yml), map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.Clean(config); err != nil {
		t.Fatal(err)
	}

	// test that images left

	all, err := dockerCli.ListImages(docker.ListImagesOptions{})
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	for _, image := range all {
		for _, repoTag := range image.RepoTags {
			imageName := NewImageNameFromString(repoTag)
			if imageName.Name == "rocker-compose-test-image-clean" {
				n++
			}
		}
	}

	assert.Equal(t, 2, n, "Expected images to be cleaned up")

	// test removed images list

	removed := cli.GetRemovedImages()
	assert.Equal(t, 3, len(removed), "Expected to remove a particular number of images")

	assert.EqualValues(t, &ImageName{"", "rocker-compose-test-image-clean", "3"}, removed[0], "removed wrong image")
	assert.EqualValues(t, &ImageName{"", "rocker-compose-test-image-clean", "2"}, removed[1], "removed wrong image")
	assert.EqualValues(t, &ImageName{"", "rocker-compose-test-image-clean", "1"}, removed[2], "removed wrong image")
}
