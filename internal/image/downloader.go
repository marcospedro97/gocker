package image

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	authURL                   = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	dockerManifestUrl         = "https://registry-1.docker.io/v2/%s/manifests/%s"
	dockerManifestByDigestURL = "https://registry-1.docker.io/v2/%s/manifests/%s"

	dockerImageUrl    = "https://registry-1.docker.io/v2/%s/blobs/%s"
	manifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
	fileExt           = ".tar.gz"
)

// STRUCTS
type AuthResponse struct {
	Token string `json:"token"`
}

type Layer struct {
	Digest string `json:"digest"`
}

type Manifest struct {
	Layers []Layer `json:"layers"`
}

type ManifestList struct {
	Manifests []PlatformDescriptor `json:"manifests"`
}

type PlatformDescriptor struct {
	Digest   string   `json:"digest"`
	Platform Platform `json:"platform"`
}

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// DownloadImage downloads a Docker image by its name and tag.
// It retrieves the authentication token, fetches the image manifest,
// and downloads each layer of the image to the specified destination directory.
// The image is identified by its name and tag, and the layers are saved as files
// in the destination directory with filenames derived from their digests.
// It returns an error if any step fails, such as authentication, manifest retrieval, or layer
func DownloadImage(imageName string, tag string, dest string) error {
	auth, err := authenticate(imageName)
	if err != nil {
		return err
	}

	fmt.Printf("Using authentication token for image %s\n", imageName)
	digest, err := selectPlatformDigest(imageName, tag, auth)
	if err != nil {
		return err
	}

	fmt.Printf("Selected digest for image %s: %s\n", imageName, digest)
	manifest, err := fetchManifest(imageName, digest, auth)
	if err != nil {
		return err
	}
	fmt.Printf("Fetched manifest for image %s with %d layers\n", imageName, len(manifest.Layers))

	for i, layer := range manifest.Layers {
		fmt.Printf("Downloading layer %d/%d: %s\n", i+1, len(manifest.Layers), layer.Digest)
		err := downloadLayer(layer.Digest, imageName, auth, dest)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Downloaded all layers for image %s to %s\n", imageName, dest)
	return nil
}

// authenticate retrieves an authentication token for the Docker registry.
// It uses the Docker Hub API to get a token that can be used to pull images.
// The token is scoped to the specified image name for pulling.
// It returns the token as a string or an error if the request fails.
func authenticate(imageName string) (string, error) {
	url := fmt.Sprintf(authURL, imageName)
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	var authResponse AuthResponse

	err = json.NewDecoder(response.Body).Decode(&authResponse)
	if err != nil {
		return "", err
	}

	return authResponse.Token, nil
}

func selectPlatformDigest(imageName, tag, token string) (string, error) {
	url := fmt.Sprintf(dockerManifestUrl, imageName, tag)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var manifestList ManifestList
	if err := json.Unmarshal(body, &manifestList); err != nil {
		return "", err
	}

	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH

	for _, m := range manifestList.Manifests {
		if m.Platform.OS == currentOS && m.Platform.Architecture == currentArch {
			return m.Digest, nil
		}
	}

	return "", fmt.Errorf("no manifest found for platform %s/%s", currentOS, currentArch)
}

// fetchManifest retrieves the manifest for a Docker image using the provided image name, tag, and authentication token.
// It constructs the URL for the manifest, sends a GET request with the token in the header,
// and decodes the response into a Manifest struct.
// It returns the Manifest struct or an error if the request fails or decoding fails.
// The manifest contains information about the image layers.
func fetchManifest(imageName string, digest string, authToken string) (Manifest, error) {
	var manifest Manifest

	manifestURL := fmt.Sprintf(dockerManifestByDigestURL, imageName, digest)
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return Manifest{}, err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", manifestMediaType)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Manifest{}, err
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	err = json.Unmarshal(body, &manifest)

	return manifest, err
}

// downloadLayer downloads a specific layer of a Docker image using its digest from the manifest.
// It constructs the URL for the layer, sends a GET request with the authentication token in the header,
// and saves the layer to a file in the specified destination directory.
// The layer is saved with a filename derived from its digest, ensuring unique identification.
// It returns an error if the request fails or if there is an issue saving the file.
func downloadLayer(digest string, imageName string, authToken string, dest string) error {
	layerURL := fmt.Sprintf(dockerImageUrl, imageName, digest)
	req, err := http.NewRequest("GET", layerURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	filePath := filepath.Join(dest, digestToFilename(digest))
	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, response.Body)
	return err
}

// digestToFilename converts a Docker image layer digest to a filename.
// It encodes the digest using URL-safe base64 encoding and appends a file extension.
// This ensures that the filename is unique and can be safely used in a filesystem.
// The resulting filename is suitable for storing the layer data in a tar.gz format.
func digestToFilename(digest string) string {
	return base64.URLEncoding.EncodeToString([]byte(digest)) + fileExt
}
