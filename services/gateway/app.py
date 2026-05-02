import logging
import os
from datetime import datetime, timezone

import httpx
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("gateway")

app = FastAPI(title="PulseGrid Gateway", version="1.0.0")

PROCESSOR_URL = os.getenv("PROCESSOR_URL", "http://localhost:8081")
DASHBOARD_URL = os.getenv("DASHBOARD_URL", "http://localhost:8082")


class PipelineEvent(BaseModel):
    pipeline_id: str
    event_type: str
    payload: dict = {}


class PipelineCreate(BaseModel):
    name: str
    source: str
    destination: str


@app.get("/health")
def health():
    return {
        "status": "healthy",
        "service": "gateway",
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


@app.get("/health/all")
async def health_all():
    results = {"gateway": "healthy"}
    async with httpx.AsyncClient(timeout=5.0) as client:
        for name, url in [("processor", PROCESSOR_URL), ("dashboard", DASHBOARD_URL)]:
            try:
                resp = await client.get(f"{url}/health")
                results[name] = resp.json().get("status", "unknown")
            except Exception as e:
                logger.error("Health check failed for %s: %s", name, e)
                results[name] = "unreachable"
    return {"services": results}


@app.post("/pipelines", status_code=201)
async def create_pipeline(pipeline: PipelineCreate):
    logger.info("Creating pipeline: %s", pipeline.name)
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.post(
                f"{PROCESSOR_URL}/pipelines",
                json=pipeline.model_dump(),
            )
            resp.raise_for_status()
            data = resp.json()
        except httpx.HTTPStatusError as e:
            logger.error("Processor error: %s", e)
            raise HTTPException(status_code=e.response.status_code, detail=str(e))
        except Exception as e:
            logger.error("Processor unreachable: %s", e)
            raise HTTPException(status_code=502, detail="Processor service unavailable")
    return data


@app.get("/pipelines")
async def list_pipelines():
    logger.info("Listing pipelines")
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(f"{PROCESSOR_URL}/pipelines")
            resp.raise_for_status()
            return resp.json()
        except Exception as e:
            logger.error("Processor unreachable: %s", e)
            raise HTTPException(status_code=502, detail="Processor service unavailable")


@app.get("/pipelines/{pipeline_id}")
async def get_pipeline(pipeline_id: str):
    logger.info("Getting pipeline: %s", pipeline_id)
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(f"{PROCESSOR_URL}/pipelines/{pipeline_id}")
            resp.raise_for_status()
            return resp.json()
        except httpx.HTTPStatusError as e:
            raise HTTPException(status_code=e.response.status_code, detail=str(e))
        except Exception as e:
            logger.error("Processor unreachable: %s", e)
            raise HTTPException(status_code=502, detail="Processor service unavailable")


@app.post("/pipelines/{pipeline_id}/events", status_code=201)
async def send_event(pipeline_id: str, event: PipelineEvent):
    logger.info("Sending event to pipeline %s: %s", pipeline_id, event.event_type)
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.post(
                f"{PROCESSOR_URL}/pipelines/{pipeline_id}/events",
                json=event.model_dump(),
            )
            resp.raise_for_status()
            return resp.json()
        except httpx.HTTPStatusError as e:
            raise HTTPException(status_code=e.response.status_code, detail=str(e))
        except Exception as e:
            logger.error("Processor unreachable: %s", e)
            raise HTTPException(status_code=502, detail="Processor service unavailable")


@app.get("/dashboard/stats")
async def dashboard_stats():
    logger.info("Fetching dashboard stats")
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(f"{DASHBOARD_URL}/stats")
            resp.raise_for_status()
            return resp.json()
        except Exception as e:
            logger.error("Dashboard unreachable: %s", e)
            raise HTTPException(status_code=502, detail="Dashboard service unavailable")
