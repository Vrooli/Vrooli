package physicalruntime

import (
	"bytes"
	"testing"

	"device-studio/internal/rendering"
)

func TestTelevisionV3RecipeReplaysPinnedThinBezelAsset(t *testing.T) {
	data, digest, err := compileTelevisionV3Asset()
	if err != nil {
		t.Fatal(err)
	}
	if digest != televisionV3AssetSHA {
		t.Fatalf("v3 recipe digest changed: %s", digest)
	}
	retained, err := starterAssets.ReadFile(televisionV3AssetFile)
	if err != nil || !bytes.Equal(retained, data) {
		t.Fatalf("embedded v3 bytes differ from deterministic recipe: read err=%v", err)
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Nodes) != len(televisionThinBezelRecipe().Parts) || inventory.Nodes[2].Name != "display" || inventory.Meshes[2].Primitives[1].UVMin != [2]float64{} || inventory.Meshes[2].Primitives[1].UVMax != [2]float64{1, 1} {
		t.Fatalf("v3 display node/primitive no longer exposes normalized app-content UVs: %+v", inventory)
	}
}
