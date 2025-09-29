const svg = d3.select("svg");
const width = +svg.attr("width");
const height = +svg.attr("height");
const radius = 300;
const centerX = width / 2;
const centerY = height / 2;

let nodes = [];
let selectedNode = null;
let keyLocations = {}

const evtSource = new EventSource("http://localhost:8080/events");
evtSource.onmessage = (event) => {
    const ev = JSON.parse(event.data);
    if (ev.type === "key") {
        console.log("Key stored:", ev.data);
    }
    if (ev.type === "stats") {
        console.log("Ring stats updated:", ev.data);

        ringSize = ev.data.size;
        nodes = ev.data.nodes.map(n => ({
            id: n.nodeID,
            hashPos: n.key,
            slots: n.slots,
            filled: n.filled
        }));

        computeNodePositions();
        drawRing();
    }
}

function animateKeyMove(key, fromNodeID, toNodeID) {
    const from = nodes.find(n => n.id === fromNodeID);
    const to = nodes.find(n => n.id === toNodeID);
    if (!from || !to) return;

    const keyCircle = svg.append("circle")
        .attr("r", 5)
        .attr("fill", "red")
        .attr("cx", from.x)
        .attr("cy", from.y);

    keyCircle.transition()
        .duration(1500)
        .attr("cx", to.x)
        .attr("cy", to.y)
        .style("opacity", 0.2)
        .remove();
}

// Fetch the ring from backend
async function fetchRing() {
  try {
        const res = await fetch("http://localhost:8080/ring"); // replace PORT
        const data = await res.json();

        const newLocations = {}
        data.nodes.forEach(node => {
            for (const k in node.keys) { 
                newLocations[k] = node.nodeID;
                if (keyLocations[k] && keyLocations[k] !== node.nodeID) {
                    animateKeyMove(k, keyLocations[k], node.nodeID)
                }
            }
        });
        keyLocations = newLocations;

        ringSize = data.size;
        // Build node objects
        nodes = data.nodes.map(node => ({
            id: node.nodeID,
            hashPos: node.key,
            slots: node.slots,
            filled: node.filled
        }));

        computeNodePositions();
        drawRing();
  } catch (err) {
      console.error("Failed to fetch ring:", err);
  }
}

// Compute x/y positions of nodes around the ring
function computeNodePositions() {
    nodes.forEach(node => {
        const angle = (node.hashPos / ringSize) * 2 * Math.PI - Math.PI / 2;
        node.x = centerX + radius * Math.cos(angle);
        node.y = centerY + radius * Math.sin(angle);
    });
}

// Draw the ring and nodes
function drawRing() {
    svg.selectAll("*").remove();

    // Ring circle
    svg.append("circle")
        .attr("cx", centerX)
        .attr("cy", centerY)
        .attr("r", radius)
        .attr("fill", "none")
        .attr("stroke", "#ccc")
        .attr("stroke-width", 2);

    // Nodes
    const nodeGroup = svg.selectAll(".node-group")
        .data(nodes, d => d.id)
        .enter()
        .append("g")
        .attr("class", "node-group")
        .attr("transform", d => `translate(${d.x},${d.y})`)
        .on("click", (event, d) => {
            d3.selectAll(".node-circle").classed("selected", false);
            d3.select(event.currentTarget).select(".node-circle").classed("selected", true);
            selectedNode = d;
            updateSidebar(d);
        });

    nodeGroup.append("circle")
        .attr("class", "node-circle")
        .attr("r", 30);

    nodeGroup.append("text")
        .attr("class", "node-label")
        .attr("text-anchor", "middle") 
        .attr("dominant-baseline", "middle")
        .text(d => d.hashPos);

    const barWidth = 100;
    const barHeight = 10;
    nodes.forEach(node => {
        const percent = node.filled / node.slots;
        svg.append("rect")
            .attr("x", node.x - barWidth/2)
            .attr("y", node.y + 35)
            .attr("width", barWidth)
            .attr("height", barHeight)
            .attr("fill", "#eee");
        svg.append("rect")
            .attr("x", node.x - barWidth/2)
            .attr("y", node.y + 35)
            .attr("width", barWidth * percent)
            .attr("height", barHeight)
            .attr("fill", "orange");
    });
}

function truncateID(id, length = 10) {
    if (id.length <= length) return id;
    const start = Math.floor(length / 2);
    const end = id.length - (length - start - 3); // 3 for "..."
    return id.slice(0, start) + "..." + id.slice(end);
}

function updateSidebar(node) {
    document.getElementById("node-id").textContent = "ID: " + truncateID(node.id, 12);
    document.getElementById("node-load").textContent = "Load: " + (node.load || 0);
    document.getElementById("node-keys").textContent = "Keys: " + node.filled;
    
    const percent = Math.round((node.filled / node.slots) * 100);
    document.getElementById("node-bar-fill").style.width = percent + "%";
    
    document.getElementById("delete-node-btn").disabled = false;
}

// Add Node
document.getElementById("add-node-btn").addEventListener("click", async () => {
    try {
        const res = await fetch("http://localhost:8080/nodes", { method: "POST" });
        if (!res.ok) throw new Error(await res.text());
        await fetchRing();
    } catch (err) {
        console.error("Failed to add node:", err);
    }
});

// Delete Node
document.getElementById("delete-node-btn").addEventListener("click", async () => {
    if (!selectedNode) return;
    try {
        const res = await fetch(`http://localhost:8080/nodes/${selectedNode.id}`, { method: "DELETE" });
        if (!res.ok) throw new Error(await res.text());
        selectedNode = null;
        document.getElementById("delete-node-btn").disabled = true;
        await fetchRing();
    } catch (err) {
        console.error("Failed to delete node:", err);
    }
});

fetchRing();
