### Consistent Hashing Demo

A hands-on demonstration of consistent hashing with Dockerized nodes, a hash ring, and real-time visualization. This project shows how keys are distributed across nodes and how the system handles node addition and removal.

#### Features

- Consistent Hashing: Distributes keys efficiently across nodes.
- Dockerized Nodes: Each node acts as a cache partition.
- API: Store and retrieve keys from specific nodes.
- Dynamic Scaling: Add or remove nodes without disrupting the system.
- Real-Time Visualization: See the hash ring, node load, and key movement with D3.

#### Getting Started

Clone the repository:

```
git clone https://github.com/higim/consistent-hashing.git
cd consistent-hashing
```

Start the system with Docker compose:

```docker-compose up --build```

The fronten visualization will be available at http://localhost:8000

#### API Endpoints

- Add node: POST /nodes
- Remove node: DELETE /nodes/:id
- Store key: POST /items
- Retrieve key: GET /items/:key

#### How it works

1. Nodes are mapped onto a hash ring.
2. Keys are assigned to nodes by moving clockwise on the ring.
3. Adding or removing nodes only reassigns the keys affected by the change.
4. The frontend shows the hash ring, node load, and key movements in real-time.
