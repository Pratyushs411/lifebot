"use client";

import { useState, ChangeEvent, FormEvent } from "react";

const API_BASE = "http://localhost:8080";

type HealthParameter = {
  name: string;
  value: string;
  unit: string;
  flag: string;
};

type AIResponse = {
  patientName: string;
  parameters: HealthParameter[];
  dietRecommendation: string;
  doctorCategory: string;
  notes: string;
};

export default function HomePage() {
  const [ai, setAi] = useState<AIResponse | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [patientName, setPatientName] = useState("");

  // Upload PDF -> /upload-report
  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append("file", file);

    try {
      setUploading(true);
      setUploadError("");
      setAi(null);

      const res = await fetch(`${API_BASE}/upload-report`, {
        method: "POST",
        body: formData,
      });

      if (!res.ok) {
        const text = await res.text().catch(() => "");
        setUploadError(
          text || "Failed to upload PDF or get recommendations from backend."
        );
        return;
      }

      const data: AIResponse = await res.json();
      setAi(data);
    } catch (err) {
      console.error(err);
      setUploadError("Network error while uploading PDF.");
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  };

  // Read Live Data -> /live-read (10s window)
  const handleReadLive = async (e: FormEvent) => {
    e.preventDefault();

    try {
      setUploading(true);
      setUploadError("");
      setAi(null);

      const res = await fetch(`${API_BASE}/live-read`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ patientName }),
      });

      if (!res.ok) {
        const text = await res.text().catch(() => "");
        setUploadError(
          text ||
            "Failed to read live data. Make sure ESP-32 is connected and sending."
        );
        return;
      }

      const data: AIResponse = await res.json();
      setAi(data);
    } catch (err) {
      console.error(err);
      setUploadError("Network error while reading live data.");
    } finally {
      setUploading(false);
    }
  };

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100 flex justify-center px-4 py-10">
      <div className="w-full max-w-4xl space-y-6">
        {/* Header */}
        <header className="text-center space-y-2">
          <h1 className="text-3xl font-bold tracking-tight">
            Lifebot Health Dashboard
          </h1>
          <p className="text-sm text-slate-400">
            Upload a lab report PDF or read live data from ESP-32 to get
            AI-based recommendations.
          </p>
        </header>

        {/* Patient name + buttons */}
        <form
          onSubmit={handleReadLive}
          className="flex flex-col md:flex-row items-center justify-center gap-3"
        >
          <input
            type="text"
            value={patientName}
            onChange={(e) => setPatientName(e.target.value)}
            placeholder="Enter patient name"
            className="w-full md:w-64 rounded-full px-4 py-2 text-sm bg-slate-900 border border-slate-700 focus:outline-none focus:ring-2 focus:ring-emerald-500"
          />

          <div className="flex gap-3 flex-wrap justify-center">
            {/* Upload PDF */}
            <label className="cursor-pointer rounded-full px-5 py-2 text-sm font-semibold bg-emerald-500 hover:bg-emerald-400 transition">
              {uploading ? "Working..." : "Upload PDF"}
              <input
                type="file"
                accept="application/pdf"
                className="hidden"
                onChange={handleFileChange}
                disabled={uploading}
              />
            </label>

            {/* Read Live Data from ESP */}
            <button
              type="submit"
              disabled={uploading}
              className="rounded-full px-5 py-2 text-sm font-semibold border border-slate-600 hover:bg-slate-800 transition disabled:opacity-50"
            >
              {uploading ? "Reading live (10s)..." : "Read Live Data (ESP-32)"}
            </button>
          </div>
        </form>

        {uploadError && (
          <p className="text-xs text-red-400 text-center mt-1">{uploadError}</p>
        )}

        {/* Patient + parameters */}
        <section className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-4">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-lg font-semibold">Patient & Parameters</h2>
          </div>

          {!ai && !uploading && (
            <p className="text-sm text-slate-400">
              Upload a PDF or read live data to see patient values.
            </p>
          )}

          {ai && (
            <>
              {/* Patient name */}
              <div className="text-sm">
                <span className="font-semibold text-slate-300">
                  Patient Name:
                </span>{" "}
                {ai.patientName || patientName || "Not specified"}
              </div>

              {/* Parameters table */}
              {ai.parameters && ai.parameters.length > 0 ? (
                <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-950/40">
                  <table className="min-w-full text-sm">
                    <thead className="bg-slate-900/80 text-slate-300 text-xs uppercase tracking-wide">
                      <tr>
                        <th className="px-3 py-2 text-left">Parameter</th>
                        <th className="px-3 py-2 text-left">Value</th>
                        <th className="px-3 py-2 text-left">Unit</th>
                        <th className="px-3 py-2 text-left">Flag</th>
                      </tr>
                    </thead>
                    <tbody>
                      {ai.parameters.map((p, idx) => (
                        <tr
                          key={idx}
                          className="border-t border-slate-800/70 hover:bg-slate-900/60"
                        >
                          <td className="px-3 py-2">{p.name}</td>
                          <td className="px-3 py-2">{p.value}</td>
                          <td className="px-3 py-2">{p.unit}</td>
                          <td className="px-3 py-2 capitalize">
                            {p.flag || "unknown"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="text-sm text-slate-400">
                  No parameters available for this reading.
                </p>
              )}
            </>
          )}
        </section>

        {/* Recommendations card */}
        <section className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-3">
          <h2 className="text-lg font-semibold">AI Recommendations</h2>

          {!ai && !uploading && (
            <p className="text-sm text-slate-400">
              Upload a report or read live data to see diet advice, doctor
              category and notes.
            </p>
          )}

          {ai && (
            <div className="space-y-3 text-sm">
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-500">
                  Diet Recommendation
                </p>
                <p className="text-slate-200 whitespace-pre-line">
                  {ai.dietRecommendation || "Not available"}
                </p>
              </div>

              <div>
                <p className="text-xs uppercase tracking-wide text-slate-500">
                  Doctor Category
                </p>
                <p className="text-slate-200">
                  {ai.doctorCategory || "Not available"}
                </p>
              </div>

              <div>
                <p className="text-xs uppercase tracking-wide text-slate-500">
                  Notes
                </p>
                <p className="text-slate-200 whitespace-pre-line">
                  {ai.notes || "No extra notes"}
                </p>
              </div>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
