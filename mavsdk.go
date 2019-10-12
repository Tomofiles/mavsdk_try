package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"google.golang.org/grpc"

	mavsdk_rpc_telemetry "mavsdk/protos/telemetry"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal("client connection error:", err)
	}
	defer conn.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mavlink := mavlink(ctx, conn)

	server := &http.Server{Addr: ":8080", Handler: nil}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	http.HandleFunc("/czml", mavlink.sse)

	go server.ListenAndServe()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	<-stop

	ctxSd, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctxSd)
}

var rwm sync.RWMutex

type Mavlink struct {
	Path []float64
	Quat []float64
}

func (mavlink *Mavlink) sse(w http.ResponseWriter, r *http.Request) {
	flusher, _ := w.(http.Flusher)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	d := time.Now().Add(-9 * time.Hour)
	docdata := Document{
		Id:      "document",
		Version: "1.0",
		Clock: &Clock{
			Interval:    d.Add(-3*time.Second).Format("2006-01-02T15:04:05Z") + "/" + d.Add(5*time.Hour).Format("2006-01-02T15:04:05Z"),
			CurrentTime: d.Add(-3 * time.Second).Format("2006-01-02T15:04:05Z"),
			Multiplier:  1.0,
			Range:       "LOOP_STOP",
			Step:        "SYSTEM_CLOCK_MULTIPLIER",
		},
	}
	jsondata, _ := json.Marshal(docdata)
	m := "id: %d\ndata: " + string(jsondata) + "\n\n"
	fmt.Fprintf(w, m, d.UnixNano())
	flusher.Flush()

	rwmLocker := rwm.RLocker()
	rwmLocker.Lock()

	d = time.Now().Add(-9 * time.Hour)
	epoch := d.Add(0 * time.Second).Format("2006-01-02T15:04:05Z")
	modeldata := ThreeDModel{
		Id:           "drone",
		Name:         "Cesium Drone",
		Availability: d.Add(0*time.Second).Format("2006-01-02T15:04:05Z") + "/" + d.Add(5*time.Hour).Format("2006-01-02T15:04:05Z"),
		Position: &Position{
			Epoch:               epoch,
			CartographicDegrees: mavlink.Path,
		},
		Orientation: &Orientation{
			Epoch:          epoch,
			UnitQuaternion: mavlink.Quat,
		},
		Path: nil,
		Model: &Model{
			Gltf:             "CesiumDrone.gltf",
			Scale:            0.5,
			MinimumPixelSize: 100,
			Show:             true,
		},
	}
	jsondata, _ = json.Marshal(modeldata)
	m = "id: %d\ndata: " + string(jsondata) + "\n\n"
	fmt.Fprintf(w, m, d.UnixNano())
	flusher.Flush()

	rwmLocker.Unlock()

	done := make(chan interface{})
	defer close(done)

	go func(done <-chan interface{}) {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()

		flightTime := 0.0

		for {
			select {
			case <-t.C:
				flightTime += 1.0

				rwmLocker = rwm.RLocker()
				rwmLocker.Lock()

				path := append([]float64{flightTime}, mavlink.Path...)
				quat := append([]float64{flightTime}, mavlink.Quat...)

				d = time.Now().Add(-9 * time.Hour)
				posdata := ThreeDModel{
					Id: "drone",
					Position: &Position{
						Epoch:               epoch,
						CartographicDegrees: path,
					},
					Orientation: &Orientation{
						Epoch:          epoch,
						UnitQuaternion: quat,
					},
					Path: &Path{
						Show:  true,
						Width: 1,
						Material: &Material{
							SolidColor: &Color{
								Rgba: []int32{0, 255, 255, 255},
							},
						},
						LeadTime: 0,
					},
					Model: nil,
				}
				jsondata, _ := json.Marshal(posdata)
				m := "id: %d\ndata: " + string(jsondata) + "\n\n"
				fmt.Fprintf(w, m, d.UnixNano())
				flusher.Flush()

				rwmLocker.Unlock()
			case <-done:
				return
			}
		}
	}(done)

	notify := w.(http.CloseNotifier).CloseNotify()
	<-notify
}

func mavlink(ctx context.Context, conn *grpc.ClientConn) *Mavlink {
	client := mavsdk_rpc_telemetry.NewTelemetryServiceClient(conn)

	positionRequest := mavsdk_rpc_telemetry.SubscribePositionRequest{}
	positionReceiver, err := client.SubscribePosition(ctx, &positionRequest)
	if err != nil {
		log.Fatal("position request error:", err)
	}

	quaternionRequest := mavsdk_rpc_telemetry.SubscribeAttitudeQuaternionRequest{}
	quaternionReceiver, err := client.SubscribeAttitudeQuaternion(ctx, &quaternionRequest)
	if err != nil {
		log.Fatal("quaternion request error:", err)
	}

	mavlink := &Mavlink{}

	go func(mavlink *Mavlink, positionReceiver mavsdk_rpc_telemetry.TelemetryService_SubscribePositionClient) <-chan mavsdk_rpc_telemetry.Position {
		positionStream := make(chan mavsdk_rpc_telemetry.Position)
		go func() {
			defer close(positionStream)
			for {
				response, err := positionReceiver.Recv()
				if err == io.EOF {
					log.Println("position response error:", err)
					return
				}
				if err != nil {
					log.Println("position response error:", err)
					return
				}
				position := response.GetPosition()
				path := []float64{}
				path = append(path, position.GetLongitudeDeg())
				path = append(path, position.GetLatitudeDeg())
				path = append(path, float64(position.GetAbsoluteAltitudeM()))

				rwm.Lock()
				mavlink.Path = path
				rwm.Unlock()

				// log.Println("position response received.")
			}
		}()
		return positionStream
	}(mavlink, positionReceiver)

	go func(mavlink *Mavlink, quaternionReceiver mavsdk_rpc_telemetry.TelemetryService_SubscribeAttitudeQuaternionClient) <-chan mavsdk_rpc_telemetry.Quaternion {
		quaternionStream := make(chan mavsdk_rpc_telemetry.Quaternion)
		go func() {
			defer close(quaternionStream)
			for {
				response, err := quaternionReceiver.Recv()
				if err == io.EOF {
					log.Println("quaternion response error:", err)
					return
				}
				if err != nil {
					log.Println("quaternion response error:", err)
					return
				}
				quaternion := response.GetAttitudeQuaternion()
				quat := []float64{}
				quat = append(quat, float64(quaternion.GetX()))
				quat = append(quat, float64(quaternion.GetY()))
				quat = append(quat, float64(quaternion.GetZ()))
				quat = append(quat, float64(quaternion.GetW()))

				rwm.Lock()
				mavlink.Quat = quat
				rwm.Unlock()

				// log.Println("quaternion response received.")
			}
		}()
		return quaternionStream
	}(mavlink, quaternionReceiver)

	return mavlink
}
