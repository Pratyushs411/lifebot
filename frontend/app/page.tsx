"use client";

import { useState, ChangeEvent } from "react";

const API_BASE = "http://localhost:8080";

type AIResponse = {
  dietRecommendation: string;
  doctorCategory: string;
  notes: string;
};

export default function HomePage() {
  const [ai, setAi] = useState<AIResponse | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [backendError, setBackendError] = useState("");

  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    // IMPORTANT: field name must be "file" to match backend
    formData.append("file", file);

    try {
      setUploading(true);
      setUploadError("");
      setBackendError("");
      setAi(null);

      const res = await fetch(`${API_BASE}/upload-report`, {
        method: "POST",
        body: formData,
      });

      if (!res.ok) {
        // try to read backend error text
        const text = await res.text().catch(() => "");
        console.error("Upload failed:", res.status, text);
        setUploadError(text || "Failed to upload PDF or get recommendations.");
        return;
      }

      const data: AIResponse = await res.json();
      setAi(data);
    } catch (err) {
      console.error(err);
      setUploadError("Network error while uploading PDF.");
    } finally {
      setUploading(false);
      // allow re-selecting same file
      e.target.value = "";
    }
  };

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100 flex justify-center px-4 py-10">
      <div className="w-full max-w-3xl space-y-6">
        <header className="text-center space-y-2">
          <h1 className="text-3xl font-bold tracking-tight">
            Lifebot Health Dashboard
          </h1>
          <p className="text-sm text-slate-400">
            Upload a lab report PDF and get AI-based diet & doctor suggestions.
          </p>
        </header>

        {/* Upload button */}
        <div className="flex justify-center">
          <label className="cursor-pointer rounded-full px-5 py-2 text-sm font-semibold bg-emerald-500 hover:bg-emerald-400 transition">
            {uploading ? "Uploading & analysing..." : "Upload PDF"}
            <input
              type="file"
              accept="application/pdf"
              className="hidden"
              onChange={handleFileChange}
              disabled={uploading}
            />
          </label>
        </div>

        {uploadError && (
          <p className="text-xs text-red-400 text-center mt-1">{uploadError}</p>
        )}
        {backendError && (
          <p className="text-xs text-red-400 text-center mt-1">
            {backendError}
          </p>
        )}

        {/* AI result card */}
        <section className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-3">
          <h2 className="text-lg font-semibold">AI Recommendations</h2>

          {!ai && !uploading && (
            <p className="text-sm text-slate-400">
              Upload a PDF report to see recommendations here.
            </p>
          )}

          {uploading && (
            <p className="text-sm text-slate-400">
              Analysing report with Gemini…
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
