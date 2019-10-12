(function () {
    "use strict";

    var viewer = new Cesium.Viewer('cesiumContainer', {
        scene3DOnly: true,
        selectionIndicator: false,
        baseLayerPicker: false,
        navigationHelpButton: false,
        homeButton: false,
        geocoder: false,
        // animation: false,
        // timeline: false,
        fullscreenButton: false
    });

    viewer.imageryLayers.remove(viewer.imageryLayers.get(0));
    viewer.imageryLayers.addImageryProvider(new Cesium.IonImageryProvider({ assetId: 2 }));

    viewer.terrainProvider = new Cesium.CesiumTerrainProvider({
        url: Cesium.IonResource.fromAssetId(1)
    });

    var czmlStream = new Cesium.CzmlDataSource();
    var czmlStreamUrl = '/czml';

    var czmlEventSource = new EventSource(czmlStreamUrl);

    czmlEventSource.onmessage = function(e) {
        var czml = JSON.parse(e.data)
        if (czml.position !== undefined && czml.orientation !== undefined) {
            var quatlocal = new Cesium.Quaternion(
                czml.orientation.unitQuaternion[1],
                czml.orientation.unitQuaternion[2],
                czml.orientation.unitQuaternion[3],
                czml.orientation.unitQuaternion[4])
            var pos = Cesium.Cartesian3.fromDegrees(
                czml.position.cartographicDegrees[1],
                czml.position.cartographicDegrees[2],
                czml.position.cartographicDegrees[3]);
            var mtx4 = Cesium.Transforms.eastNorthUpToFixedFrame(pos);
            var mtx3 = Cesium.Matrix4.getMatrix3(mtx4, new Cesium.Matrix3());
            var base = Cesium.Quaternion.fromRotationMatrix(mtx3)
            var quat = Cesium.Quaternion.multiply(base, quatlocal, new Cesium.Quaternion())
            czml.orientation.unitQuaternion[1] = quat.x;
            czml.orientation.unitQuaternion[2] = quat.y;
            czml.orientation.unitQuaternion[3] = quat.z;
            czml.orientation.unitQuaternion[4] = quat.w;
        }
        czmlStream.process(czml);
    }

    viewer.dataSources.add(czmlStream);

    viewer.clock.shouldAnimate = true;
    var initialPosition = new Cesium.Cartesian3.fromDegrees(-73.981630, 40.731474, 8000);
    var homeCameraView = {
        destination : initialPosition,
    };
    viewer.scene.camera.setView(homeCameraView);
}());