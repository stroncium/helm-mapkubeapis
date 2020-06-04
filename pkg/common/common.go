/*
Copyright

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	utils "github.com/maorfr/helm-plugin-utils/pkg"
	"github.com/pkg/errors"

	"github.com/hickeyma/helm-mapkubeapis/pkg/mapping"
)

// KubeConfig are the Kubernetes configuration settings
type KubeConfig struct {
	Context string
	File    string
}

// MapOptions are the options for mapping deprecated APIs in a release
type MapOptions struct {
	DryRun           bool
	KubeConfig       KubeConfig
	MapFile          string
	ReleaseName      string
	ReleaseNamespace string
	StorageType      string
	TillerOutCluster bool
}

// UpgradeDescription is description of why release was upgraded
const UpgradeDescription = "Kubernetes deprecated API upgrade - DO NOT rollback from this version"

// ReplaceManifestUnSupportedAPIs returns a release manifest with deprecated or removed
// Kubernetes APIs updated to supported APIs
func ReplaceManifestUnSupportedAPIs(origManifest, mapFile string, kubeConfig KubeConfig) (string, error) {
	var modifiedManifest = origManifest
	var err error
	var mapMetadata *mapping.Metadata

	// Load the mapping data
	if mapMetadata, err = mapping.LoadMapfile(mapFile); err != nil {
		return "", errors.Wrapf(err, "Failed to load mapping file: %s", mapFile)
	}

	// get the Kubernetes server version
	kubeVersionStr, err := getKubernetesServerVersion(kubeConfig)
	if err != nil {
		return "", err
	}
	kubeVersion, err := strconv.ParseFloat(kubeVersionStr, 32)
	if err != nil {
		errors.Wrap(err, "Failed to get Kubernetes server version")
	}

	log.Printf("Kubernetes server version: '%s', parsed as %f", kubeVersionStr, kubeVersion)

	// Check for deprecated or removed APIs and map accordingly to supported versions
	for _, mapping := range mapMetadata.Mappings {
		deprecatedAPI := mapping.DeprecatedAPI
		supportedAPI := mapping.NewAPI
		var apiVersionStr string
		if mapping.DeprecatedInVersion != "" {
			apiVersionStr = mapping.DeprecatedInVersion
		} else {
			apiVersionStr = mapping.RemovedInVersion
		}
		apiVersion, err := strconv.ParseFloat(apiVersionStr, 32)
		if err != nil {
			errors.Wrapf(err, "Failed to get the deprecated or removed Kubernetes version for API: %s", deprecatedAPI)
		}

		var modManifestForAPI string
		var modified = false
		modManifestForAPI = strings.ReplaceAll(modifiedManifest, deprecatedAPI, supportedAPI)
		if modManifestForAPI != modifiedManifest {
			modified = true
			log.Printf("Found deprecated or removed Kubernetes API:\n\"%s\"\nSupported API equivalent:\n\"%s\"\n", deprecatedAPI, supportedAPI)
		}
		if modified {
			if apiVersion > kubeVersion {
				log.Printf("The following API does not require mapping now as it is not valid till Kubernetes '%s'(current:'%s') :\n\"%s\"\n", apiVersionStr, kubeVersionStr,
					deprecatedAPI)
			} else {
				modifiedManifest = modManifestForAPI
			}
		}
	}

	return modifiedManifest, nil
}

func getKubernetesServerVersion(kubeConfig KubeConfig) (string, error) {
	clientSet := utils.GetClientSetWithKubeConfig(kubeConfig.File, kubeConfig.Context)
	if clientSet == nil {
		return "", errors.Errorf("kubernetes cluster unreachable")
	}
	kubeVersion, err := clientSet.ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "kubernetes cluster unreachable")
	}
	return fmt.Sprintf("%s.%s", kubeVersion.Major, kubeVersion.Minor), nil
}
