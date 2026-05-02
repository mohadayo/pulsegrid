import express, { Request, Response } from "express";

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 8082;
const LOG_LEVEL = process.env.LOG_LEVEL || "info";

function log(level: string, message: string): void {
  if (level === "debug" && LOG_LEVEL !== "debug") return;
  const timestamp = new Date().toISOString();
  console.log(`${timestamp} [${level.toUpperCase()}] dashboard: ${message}`);
}

interface MetricEntry {
  pipeline_id: string;
  metric_name: string;
  value: number;
  timestamp: string;
}

interface Alert {
  id: string;
  pipeline_id: string;
  severity: "low" | "medium" | "high" | "critical";
  message: string;
  created_at: string;
  resolved: boolean;
}

const metrics: MetricEntry[] = [];
const alerts: Alert[] = [];
let alertCounter = 0;

app.get("/health", (_req: Request, res: Response) => {
  res.json({
    status: "healthy",
    service: "dashboard",
    timestamp: new Date().toISOString(),
  });
});

app.get("/stats", (_req: Request, res: Response) => {
  log("info", "Fetching stats");

  const pipelineIds = [...new Set(metrics.map((m) => m.pipeline_id))];
  const totalMetrics = metrics.length;
  const activeAlerts = alerts.filter((a) => !a.resolved).length;

  const statsByPipeline: Record<string, { count: number; latest_value: number }> = {};
  for (const m of metrics) {
    if (!statsByPipeline[m.pipeline_id]) {
      statsByPipeline[m.pipeline_id] = { count: 0, latest_value: 0 };
    }
    statsByPipeline[m.pipeline_id].count++;
    statsByPipeline[m.pipeline_id].latest_value = m.value;
  }

  res.json({
    total_pipelines: pipelineIds.length,
    total_metrics: totalMetrics,
    active_alerts: activeAlerts,
    pipelines: statsByPipeline,
  });
});

app.post("/metrics", (req: Request, res: Response) => {
  const { pipeline_id, metric_name, value } = req.body;

  if (!pipeline_id || !metric_name || value === undefined) {
    res.status(400).json({ error: "pipeline_id, metric_name, and value are required" });
    return;
  }

  const entry: MetricEntry = {
    pipeline_id,
    metric_name,
    value,
    timestamp: new Date().toISOString(),
  };

  metrics.push(entry);
  log("info", `Metric recorded: ${metric_name}=${value} for pipeline ${pipeline_id}`);

  if (metric_name === "error_rate" && value > 0.5) {
    const alert: Alert = {
      id: `alert-${++alertCounter}`,
      pipeline_id,
      severity: value > 0.8 ? "critical" : "high",
      message: `High error rate detected: ${value}`,
      created_at: new Date().toISOString(),
      resolved: false,
    };
    alerts.push(alert);
    log("info", `Alert created: ${alert.id} - ${alert.message}`);
  }

  res.status(201).json(entry);
});

app.get("/metrics", (req: Request, res: Response) => {
  const { pipeline_id } = req.query;
  log("info", `Listing metrics${pipeline_id ? ` for pipeline ${pipeline_id}` : ""}`);

  const filtered = pipeline_id
    ? metrics.filter((m) => m.pipeline_id === pipeline_id)
    : metrics;

  res.json(filtered);
});

app.get("/alerts", (req: Request, res: Response) => {
  const { resolved } = req.query;
  log("info", "Listing alerts");

  let filtered = alerts;
  if (resolved === "true") {
    filtered = alerts.filter((a) => a.resolved);
  } else if (resolved === "false") {
    filtered = alerts.filter((a) => !a.resolved);
  }

  res.json(filtered);
});

app.patch("/alerts/:id/resolve", (req: Request, res: Response) => {
  const alert = alerts.find((a) => a.id === req.params.id);
  if (!alert) {
    res.status(404).json({ error: "alert not found" });
    return;
  }

  alert.resolved = true;
  log("info", `Alert resolved: ${alert.id}`);
  res.json(alert);
});

export { app };

if (require.main === module) {
  app.listen(PORT, () => {
    log("info", `Dashboard API started on port ${PORT}`);
  });
}
