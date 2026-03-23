const shell = document.querySelector(".shell-join");

if (shell) {
  const video = document.getElementById("camera");
  const preview = document.getElementById("preview");
  const previewFrame = document.getElementById("previewFrame");
  const startCameraButton = document.getElementById("startCamera");
  const captureButton = document.getElementById("capturePhoto");
  const retakeButton = document.getElementById("retakePhoto");
  const uploadInput = document.getElementById("uploadInput");
  const uploadForm = document.getElementById("uploadForm");
  const publishButton = document.getElementById("publishPhoto");
  const statusMessage = document.getElementById("statusMessage");

  const previewContext = preview.getContext("2d");
  let mediaStream = null;
  let sourceImage = null;

  function setStatus(message) {
    statusMessage.textContent = message;
  }

  async function startCamera() {
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "user" },
        audio: false,
      });
      video.srcObject = mediaStream;
      captureButton.disabled = false;
      setStatus("Camera ready. Capture a photo when the frame looks good.");
    } catch (error) {
      console.error(error);
      setStatus("Camera access failed. You can still upload an image file.");
    }
  }

  function getSettings() {
    const formData = new FormData(uploadForm);
    return {
      brightness: Number(formData.get("brightness")) / 100,
      contrast: Number(formData.get("contrast")) / 100,
      saturation: Number(formData.get("saturation")) / 100,
      grayscale: Number(formData.get("grayscale")) / 100,
      tone: String(formData.get("tone")),
      backdrop: String(formData.get("backdrop")),
    };
  }

  function backdropGradient(backdrop) {
    switch (backdrop) {
      case "mint":
        return ["#d8f3dc", "#95d5b2"];
      case "night":
        return ["#0b132b", "#3a506b"];
      case "paper":
        return ["#f8edeb", "#e8e8e4"];
      default:
        return ["#f97316", "#fb7185"];
    }
  }

  function applyTone() {
    const { tone } = getSettings();
    switch (tone) {
      case "warm":
        return { tint: "rgba(255, 140, 66, 0.12)" };
      case "cool":
        return { tint: "rgba(69, 123, 157, 0.12)" };
      case "party":
        return { tint: "rgba(168, 85, 247, 0.12)" };
      default:
        return { tint: "transparent" };
    }
  }

  function drawPreview() {
    if (!sourceImage) {
      return;
    }

    const { brightness, contrast, saturation, grayscale, backdrop } = getSettings();
    const [bgStart, bgEnd] = backdropGradient(backdrop);
    const { tint } = applyTone();

    const size = 1080;
    preview.width = size;
    preview.height = size;

    const gradient = previewContext.createLinearGradient(0, 0, size, size);
    gradient.addColorStop(0, bgStart);
    gradient.addColorStop(1, bgEnd);
    previewContext.fillStyle = gradient;
    previewContext.fillRect(0, 0, size, size);

    const fitScale = Math.max(size / sourceImage.width, size / sourceImage.height);
    const drawWidth = sourceImage.width * fitScale;
    const drawHeight = sourceImage.height * fitScale;
    const offsetX = (size - drawWidth) / 2;
    const offsetY = (size - drawHeight) / 2;

    previewContext.save();
    previewContext.filter = `brightness(${brightness}) contrast(${contrast}) saturate(${saturation}) grayscale(${grayscale})`;
    previewContext.drawImage(sourceImage, offsetX, offsetY, drawWidth, drawHeight);
    previewContext.restore();

    previewContext.fillStyle = tint;
    previewContext.fillRect(0, 0, size, size);

    preview.hidden = false;
    previewFrame.hidden = false;
    video.hidden = true;
    publishButton.disabled = false;
  }

  function makeImageFromCanvas(sourceCanvas) {
    const img = new Image();
    img.onload = () => {
      sourceImage = img;
      drawPreview();
    };
    img.src = sourceCanvas.toDataURL("image/jpeg", 0.92);
  }

  function captureFromVideo() {
    if (!video.videoWidth || !video.videoHeight) {
      setStatus("Camera is not ready yet.");
      return;
    }

    const tempCanvas = document.createElement("canvas");
    tempCanvas.width = video.videoWidth;
    tempCanvas.height = video.videoHeight;
    const ctx = tempCanvas.getContext("2d");
    ctx.drawImage(video, 0, 0);
    makeImageFromCanvas(tempCanvas);
    retakeButton.disabled = false;
    setStatus("Preview ready. Adjust the look and publish when ready.");
  }

  function loadFile(file) {
    if (!file) {
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      const img = new Image();
      img.onload = () => {
        sourceImage = img;
        drawPreview();
        retakeButton.disabled = false;
        setStatus("Image loaded. Adjust the look and publish when ready.");
      };
      img.src = reader.result;
    };
    reader.readAsDataURL(file);
  }

  async function publishPhoto(event) {
    event.preventDefault();
    if (!sourceImage) {
      setStatus("Capture or upload a photo first.");
      return;
    }

    publishButton.disabled = true;
    setStatus("Publishing to the board...");

    const caption = new FormData(uploadForm).get("caption");
    const settings = getSettings();
    const filterLabel = `${settings.tone}/${settings.backdrop}`;

    const blob = await new Promise((resolve) => preview.toBlob(resolve, "image/jpeg", 0.85));
    const formData = new FormData();
    formData.append("photo", blob, "board-photo.jpg");
    formData.append("caption", caption || "");
    formData.append("filterLabel", filterLabel);

    try {
      const response = await fetch("/api/photos", {
        method: "POST",
        body: formData,
      });
      if (!response.ok) {
        throw new Error("Upload failed");
      }
      setStatus("Published. Your photo should appear on the board immediately.");
    } catch (error) {
      console.error(error);
      setStatus("Upload failed. Please try again.");
    } finally {
      publishButton.disabled = false;
    }
  }

  function retakePhoto() {
    sourceImage = null;
    preview.hidden = true;
    previewFrame.hidden = true;
    video.hidden = false;
    publishButton.disabled = true;
    retakeButton.disabled = true;
    setStatus("Ready for another photo.");
  }

  startCameraButton?.addEventListener("click", startCamera);
  captureButton?.addEventListener("click", captureFromVideo);
  retakeButton?.addEventListener("click", retakePhoto);
  uploadInput?.addEventListener("change", (event) => {
    loadFile(event.target.files?.[0] || null);
  });
  uploadForm?.addEventListener("input", () => {
    if (sourceImage) {
      drawPreview();
    }
  });
  uploadForm?.addEventListener("submit", publishPhoto);
}
