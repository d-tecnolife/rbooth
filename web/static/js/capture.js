const shell = document.querySelector(".shell-capture");

if (shell) {
  const appName = String(document.body?.dataset.appName || "rbooth").toLowerCase().replace(/[^a-z0-9]+/g, "-");
  const DRAFT_STORAGE_KEY = `${appName || "rbooth"}:capture-draft`;
  const captureAssetsNode = document.getElementById("captureAssetsData");
  const captureAssets = captureAssetsNode
    ? JSON.parse(captureAssetsNode.textContent || '{"backdrops":{},"frames":{}}')
    : { backdrops: {}, frames: {} };
  const assetImageCache = new Map();
  const captureEditor = document.querySelector("[data-capture-editor]");
  const toggleEditorButton = document.getElementById("toggleEditor");
  const preview = document.getElementById("preview");
  const previewFrame = document.getElementById("previewFrame");
  const previewPlaceholder = document.getElementById("previewPlaceholder");
  const segmentationOverlay = document.getElementById("segmentationOverlay");
  const pickPhotoButton = document.getElementById("pickPhoto");
  const photoInput = document.getElementById("photoInput");
  const uploadForm = document.getElementById("uploadForm");
  const resetEditorButton = document.getElementById("resetEditor");
  const publishButton = document.getElementById("publishPhoto");
  const statusMessage = document.getElementById("statusMessage");

  const previewContext = preview.getContext("2d");
  const sliderOutputs = new Map(
    Array.from(uploadForm.querySelectorAll("[data-slider-value]")).map((node) => [node.dataset.sliderValue, node]),
  );
  const state = {
    sourceImage: null,
    sourceDataURL: "",
    subjectImage: null,
    segmentationModel: null,
    segmentationTasks: null,
    segmentationPreloadStarted: false,
    segmentationPending: false,
    segmentationSourceKey: "",
    submitting: false,
  };

  function setStatus(message) {
    statusMessage.textContent = message;
  }

  function saveDraft() {
    try {
      const draft = {
        sourceDataURL: state.sourceDataURL || "",
        caption: String(new FormData(uploadForm).get("caption") || ""),
        settings: getSettings(),
      };
      sessionStorage.setItem(DRAFT_STORAGE_KEY, JSON.stringify(draft));
    } catch (error) {
      console.error(error);
    }
  }

  function clearDraft() {
    try {
      sessionStorage.removeItem(DRAFT_STORAGE_KEY);
    } catch (error) {
      console.error(error);
    }
  }

  function syncSegmentationOverlay() {
    const shouldShow =
      Boolean(state.sourceImage) &&
      state.segmentationPending &&
      getSettings().backdrop !== "none";
    segmentationOverlay.hidden = !shouldShow;
    publishButton.disabled = !state.sourceImage || state.segmentationPending || state.submitting;
  }

  function syncPreviewPlaceholder() {
    const hasImage = Boolean(state.sourceImage);
    previewPlaceholder.hidden = hasImage;
    previewPlaceholder.style.display = hasImage ? "none" : "";
    pickPhotoButton.hidden = false;
    pickPhotoButton.style.display = "";
    preview.hidden = !hasImage;
    previewFrame.hidden = !hasImage;
    syncSegmentationOverlay();
  }

  function setEditorOpen(isOpen) {
    captureEditor?.classList.toggle("capture-editor-open", isOpen);
    if (toggleEditorButton) {
      toggleEditorButton.setAttribute("aria-expanded", String(isOpen));
      toggleEditorButton.setAttribute("aria-label", isOpen ? "Hide edit controls" : "Open edit controls");
    }
  }

  function getSettings() {
    const formData = new FormData(uploadForm);
    return {
      brightness: Number(formData.get("brightness")),
      contrast: Number(formData.get("contrast")),
      saturation: Number(formData.get("saturation")),
      grayscale: Number(formData.get("grayscale")),
      tone: String(formData.get("tone")),
      backdrop: String(formData.get("backdrop")),
      frame: String(formData.get("frame")),
    };
  }

  function selectedAssetURL(kind, value) {
    return String(captureAssets?.[kind]?.[value] || "");
  }

  function loadAssetImage(url) {
    if (!url) {
      return Promise.resolve(null);
    }
    if (assetImageCache.has(url)) {
      return assetImageCache.get(url);
    }

    const promise = new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error(`failed to load asset ${url}`));
      image.src = url;
    });
    assetImageCache.set(url, promise);
    return promise;
  }

  function updateSliderValues() {
    const settings = getSettings();
    sliderOutputs.get("brightness").textContent = `${settings.brightness}%`;
    sliderOutputs.get("contrast").textContent = `${settings.contrast}%`;
    sliderOutputs.get("saturation").textContent = `${settings.saturation}%`;
    sliderOutputs.get("grayscale").textContent = `${settings.grayscale}%`;
  }

  function backgroundPalette(backdrop) {
    switch (backdrop) {
      case "none":
        return null;
      case "sunrise":
        return {
          colors: ["#f97316", "#fb7185"],
          accent: "rgba(255, 244, 251, 0.22)",
        };
      case "mint":
        return {
          colors: ["#d8f3dc", "#95d5b2"],
          accent: "rgba(58, 180, 140, 0.18)",
        };
      case "night":
        return {
          colors: ["#1d3557", "#3d5a80"],
          accent: "rgba(244, 114, 182, 0.16)",
        };
      case "paper":
        return {
          colors: ["#f8edeb", "#e8e8e4"],
          accent: "rgba(186, 164, 143, 0.15)",
        };
      case "studio":
        return {
          colors: ["#f1f5f9", "#dbeafe"],
          accent: "rgba(236, 72, 153, 0.14)",
        };
      default:
        return null;
    }
  }

  function toneOverlay(tone) {
    switch (tone) {
      case "warm":
        return "rgba(255, 166, 117, 0.18)";
      case "cool":
        return "rgba(112, 161, 255, 0.16)";
      case "party":
        return "rgba(212, 119, 255, 0.16)";
      default:
        return "transparent";
    }
  }

  function drawRoundedRect(ctx, x, y, width, height, radius) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
    ctx.closePath();
  }

  function drawBackdrop(width, height, palette) {
    const gradient = previewContext.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, palette.colors[0]);
    gradient.addColorStop(1, palette.colors[1]);
    previewContext.fillStyle = gradient;
    previewContext.fillRect(0, 0, width, height);

    previewContext.fillStyle = palette.accent;
    previewContext.beginPath();
    previewContext.arc(width * 0.22, height * 0.24, Math.min(width, height) * 0.16, 0, Math.PI * 2);
    previewContext.fill();
    previewContext.beginPath();
    previewContext.arc(width * 0.76, height * 0.18, Math.min(width, height) * 0.12, 0, Math.PI * 2);
    previewContext.fill();
    previewContext.beginPath();
    previewContext.arc(width * 0.78, height * 0.8, Math.min(width, height) * 0.2, 0, Math.PI * 2);
    previewContext.fill();
  }

  function clampChannel(value) {
    return Math.max(0, Math.min(255, value));
  }

  async function getSegmentationModel() {
    if (state.segmentationModel) {
      return state.segmentationModel;
    }

    if (!state.segmentationTasks) {
      state.segmentationTasks = import("https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.14/+esm");
    }

    const { FilesetResolver, ImageSegmenter } = await state.segmentationTasks;
    const vision = await FilesetResolver.forVisionTasks(
      "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.14/wasm",
    );
    const model = await ImageSegmenter.createFromOptions(vision, {
      baseOptions: {
        modelAssetPath:
          "https://storage.googleapis.com/mediapipe-models/image_segmenter/selfie_multiclass_256x256/float32/latest/selfie_multiclass_256x256.tflite",
      },
      runningMode: "IMAGE",
      outputConfidenceMasks: true,
      outputCategoryMask: false,
    });
    state.segmentationModel = model;
    return model;
  }

  function preloadSegmentationModel() {
    if (state.segmentationPreloadStarted) {
      return;
    }
    state.segmentationPreloadStarted = true;
    getSegmentationModel().catch((error) => {
      console.error(error);
    });
  }

  function buildSubjectCanvas(image, confidenceMask) {
    const width = image.naturalWidth || image.width;
    const height = image.naturalHeight || image.height;
    const maskWidth = confidenceMask.width;
    const maskHeight = confidenceMask.height;
    const backgroundMask = confidenceMask.getAsFloat32Array();
    const maskCanvas = document.createElement("canvas");
    maskCanvas.width = maskWidth;
    maskCanvas.height = maskHeight;
    const maskContext = maskCanvas.getContext("2d");
    const maskImageData = maskContext.createImageData(maskWidth, maskHeight);
    const maskPixels = maskImageData.data;

    const lowerBound = 0.5;
    const upperBound = 0.78;
    for (let index = 0; index < backgroundMask.length; index += 1) {
      const foreground = 1 - backgroundMask[index];
      let normalized = 0;
      if (foreground >= upperBound) {
        normalized = 1;
      } else if (foreground > lowerBound) {
        normalized = (foreground - lowerBound) / (upperBound - lowerBound);
      }
      const alpha = Math.max(0, Math.min(255, Math.round(normalized * 255)));
      const pixelIndex = index * 4;
      maskPixels[pixelIndex] = 255;
      maskPixels[pixelIndex + 1] = 255;
      maskPixels[pixelIndex + 2] = 255;
      maskPixels[pixelIndex + 3] = alpha < 24 ? 0 : alpha > 242 ? 255 : alpha;
    }
    maskContext.putImageData(maskImageData, 0, 0);

    const scaledMaskCanvas = document.createElement("canvas");
    scaledMaskCanvas.width = width;
    scaledMaskCanvas.height = height;
    const scaledMaskContext = scaledMaskCanvas.getContext("2d");
    scaledMaskContext.imageSmoothingEnabled = true;
    scaledMaskContext.drawImage(maskCanvas, 0, 0, width, height);
    const maskData = scaledMaskContext.getImageData(0, 0, width, height).data;

    const subjectCanvas = document.createElement("canvas");
    subjectCanvas.width = width;
    subjectCanvas.height = height;
    const subjectContext = subjectCanvas.getContext("2d");
    subjectContext.drawImage(image, 0, 0, width, height);
    const subjectImageData = subjectContext.getImageData(0, 0, width, height);
    const pixels = subjectImageData.data;

    for (let index = 0; index < pixels.length; index += 4) {
      pixels[index + 3] = maskData[index + 3];
    }

    subjectContext.putImageData(subjectImageData, 0, 0);
    return subjectCanvas;
  }

  async function segmentSubject() {
    if (!state.sourceImage || state.segmentationPending) {
      return;
    }

    const settings = getSettings();
    if (settings.backdrop === "none") {
      state.subjectImage = null;
      state.segmentationSourceKey = "";
      state.segmentationPending = false;
      drawPreview();
      return;
    }

    const sourceKey = state.sourceImage.src || `${state.sourceImage.width}x${state.sourceImage.height}`;
    if (state.subjectImage && state.segmentationSourceKey === sourceKey) {
      return;
    }

    const model = await getSegmentationModel();
    if (!model) {
      state.subjectImage = null;
      drawPreview();
      return;
    }

    state.segmentationPending = true;
    syncSegmentationOverlay();
    setStatus("Separating the subject from the background...");

    try {
      const result = model.segment(state.sourceImage);
      if (!result?.confidenceMasks?.length) {
        throw new Error("No confidence mask returned");
      }
      const subjectCanvas = buildSubjectCanvas(state.sourceImage, result.confidenceMasks[0]);
      state.subjectImage = subjectCanvas;
      state.segmentationSourceKey = sourceKey;
      drawPreview();
      setStatus("Photo ready. Adjust the look and publish when you are happy with it.");
    } catch (error) {
      console.error(error);
      state.subjectImage = null;
      state.segmentationSourceKey = "";
      drawPreview();
      setStatus("Photo ready. Background replacement is unavailable on this device, but the editor still works.");
    } finally {
      state.segmentationPending = false;
      syncSegmentationOverlay();
    }
  }

  function applyImageAdjustments(sourceCanvas, settings) {
    const context = sourceCanvas.getContext("2d");
    const imageData = context.getImageData(0, 0, sourceCanvas.width, sourceCanvas.height);
    const pixels = imageData.data;
    const brightness = settings.brightness / 100;
    const saturation = settings.saturation / 100;
    const grayscale = settings.grayscale / 100;
    const contrastAmount = settings.contrast - 100;
    const contrastFactor = (259 * (contrastAmount + 255)) / (255 * (259 - contrastAmount || 1));

    for (let index = 0; index < pixels.length; index += 4) {
      let red = pixels[index];
      let green = pixels[index + 1];
      let blue = pixels[index + 2];

      red *= brightness;
      green *= brightness;
      blue *= brightness;

      red = contrastFactor * (red - 128) + 128;
      green = contrastFactor * (green - 128) + 128;
      blue = contrastFactor * (blue - 128) + 128;

      const luminance = 0.2126 * red + 0.7152 * green + 0.0722 * blue;
      red = luminance + (red - luminance) * saturation;
      green = luminance + (green - luminance) * saturation;
      blue = luminance + (blue - luminance) * saturation;

      red = red * (1 - grayscale) + luminance * grayscale;
      green = green * (1 - grayscale) + luminance * grayscale;
      blue = blue * (1 - grayscale) + luminance * grayscale;

      pixels[index] = clampChannel(red);
      pixels[index + 1] = clampChannel(green);
      pixels[index + 2] = clampChannel(blue);
    }

    context.putImageData(imageData, 0, 0);
  }

  function drawFrameDecor(frame, x, y, width, height) {
    if (frame === "none") {
      return;
    }

    previewContext.save();
    const outerX = x - 10;
    const outerY = y - 10;
    const outerWidth = width + 20;
    const outerHeight = height + 20;

    if (frame === "classic") {
      previewContext.strokeStyle = "rgba(255,255,255,0.95)";
      previewContext.lineWidth = 18;
      drawRoundedRect(previewContext, outerX, outerY, outerWidth, outerHeight, 22);
      previewContext.stroke();
      previewContext.strokeStyle = "rgba(237, 183, 206, 0.95)";
      previewContext.lineWidth = 4;
      drawRoundedRect(previewContext, outerX + 10, outerY + 10, outerWidth - 20, outerHeight - 20, 14);
      previewContext.stroke();
    }

    if (frame === "polaroid") {
      const polaroidX = x - 22;
      const polaroidY = y - 22;
      const polaroidWidth = width + 44;
      const polaroidHeight = height + 92;
      const windowX = x - 1;
      const windowY = y - 1;
      const windowWidth = width + 2;
      const windowHeight = height + 2;

      previewContext.beginPath();
      drawRoundedRect(previewContext, polaroidX, polaroidY, polaroidWidth, polaroidHeight, 18);
      drawRoundedRect(previewContext, windowX, windowY, windowWidth, windowHeight, 10);
      previewContext.fillStyle = "rgba(255,255,255,0.96)";
      previewContext.fill("evenodd");
      previewContext.strokeStyle = "rgba(234, 198, 214, 0.9)";
      previewContext.lineWidth = 2;
      drawRoundedRect(previewContext, polaroidX, polaroidY, polaroidWidth, polaroidHeight, 18);
      previewContext.stroke();
    }

    if (frame === "sparkle") {
      previewContext.strokeStyle = "rgba(255, 248, 255, 0.95)";
      previewContext.lineWidth = 14;
      drawRoundedRect(previewContext, outerX, outerY, outerWidth, outerHeight, 20);
      previewContext.stroke();
      previewContext.fillStyle = "rgba(255,255,255,0.95)";
      [
        [outerX + 12, outerY + 12],
        [outerX + outerWidth - 12, outerY + 20],
        [outerX + 18, outerY + outerHeight - 20],
        [outerX + outerWidth - 20, outerY + outerHeight - 14],
      ].forEach(([sparkX, sparkY]) => {
        previewContext.beginPath();
        previewContext.moveTo(sparkX, sparkY - 10);
        previewContext.lineTo(sparkX + 4, sparkY - 4);
        previewContext.lineTo(sparkX + 10, sparkY);
        previewContext.lineTo(sparkX + 4, sparkY + 4);
        previewContext.lineTo(sparkX, sparkY + 10);
        previewContext.lineTo(sparkX - 4, sparkY + 4);
        previewContext.lineTo(sparkX - 10, sparkY);
        previewContext.lineTo(sparkX - 4, sparkY - 4);
        previewContext.closePath();
        previewContext.fill();
      });
    }

    if (frame === "ticket") {
      previewContext.strokeStyle = "rgba(255, 249, 252, 0.95)";
      previewContext.lineWidth = 12;
      previewContext.setLineDash([12, 10]);
      drawRoundedRect(previewContext, outerX, outerY, outerWidth, outerHeight, 18);
      previewContext.stroke();
      previewContext.setLineDash([]);
    }

    previewContext.restore();
  }

  async function drawPreview() {
    if (!state.sourceImage) {
      syncPreviewPlaceholder();
      return;
    }

    const settings = getSettings();
    const backdropAssetURL = selectedAssetURL("backdrops", settings.backdrop);
    const frameAssetURL = selectedAssetURL("frames", settings.frame);
    const palette = backgroundPalette(settings.backdrop);
    const overlay = toneOverlay(settings.tone);
    const useBackgroundReplacement = settings.backdrop !== "none";
    const waitingForSegmentation = useBackgroundReplacement && state.segmentationPending && !state.subjectImage;
    const baseWidth = state.sourceImage.naturalWidth || state.sourceImage.width;
    const baseHeight = state.sourceImage.naturalHeight || state.sourceImage.height;
    const renderSource = useBackgroundReplacement && state.subjectImage ? state.subjectImage : state.sourceImage;

    preview.width = baseWidth;
    preview.height = baseHeight;
    preview.parentElement.style.setProperty("--preview-aspect", `${baseWidth} / ${baseHeight}`);
    previewContext.clearRect(0, 0, baseWidth, baseHeight);
    if (!waitingForSegmentation) {
      if (backdropAssetURL) {
        try {
          const backdropImage = await loadAssetImage(backdropAssetURL);
          if (backdropImage) {
            previewContext.drawImage(backdropImage, 0, 0, baseWidth, baseHeight);
          }
        } catch (error) {
          console.error(error);
          setStatus("could not load the selected background asset.");
        }
      } else if (palette) {
        drawBackdrop(baseWidth, baseHeight, palette);
      }
    }

    const outerPadX = palette && !waitingForSegmentation ? Math.round(baseWidth * 0.018) : 0;
    const outerPadY = palette && !waitingForSegmentation ? Math.round(baseHeight * 0.018) : 0;
    const frameX = outerPadX;
    const frameY = outerPadY;
    const frameWidth = baseWidth - outerPadX * 2;
    const frameHeight = baseHeight - outerPadY * 2;

    previewContext.save();
    previewContext.shadowColor = "rgba(89, 46, 70, 0.18)";
    previewContext.shadowBlur = 34;
    previewContext.shadowOffsetY = 18;
    if (palette && !waitingForSegmentation) {
      previewContext.strokeStyle = "rgba(255, 255, 255, 0.42)";
      previewContext.lineWidth = 2;
      drawRoundedRect(previewContext, frameX, frameY, frameWidth, frameHeight, 34);
      previewContext.stroke();
    }
    previewContext.restore();

    const photoPadding = 0;
    const photoX = frameX + photoPadding;
    const photoY = frameY + photoPadding;
    const photoWidth = frameWidth - photoPadding * 2;
    const photoHeight = frameHeight - photoPadding * 2;
    const sourceWidth = renderSource.naturalWidth || renderSource.width;
    const sourceHeight = renderSource.naturalHeight || renderSource.height;
    const fitScale = Math.min(photoWidth / sourceWidth, photoHeight / sourceHeight);
    const drawWidth = sourceWidth * fitScale;
    const drawHeight = sourceHeight * fitScale;
    const offsetX = photoX + (photoWidth - drawWidth) / 2;
    const offsetY = photoY + (photoHeight - drawHeight) / 2;

    const photoCanvas = document.createElement("canvas");
    photoCanvas.width = Math.max(1, Math.round(photoWidth));
    photoCanvas.height = Math.max(1, Math.round(photoHeight));
    const photoContext = photoCanvas.getContext("2d");
    photoContext.drawImage(
      renderSource,
      (photoWidth - drawWidth) / 2,
      (photoHeight - drawHeight) / 2,
      drawWidth,
      drawHeight,
    );
    applyImageAdjustments(photoCanvas, settings);

    previewContext.save();
    drawRoundedRect(previewContext, photoX, photoY, photoWidth, photoHeight, 24);
    previewContext.clip();
    previewContext.drawImage(photoCanvas, photoX, photoY, photoWidth, photoHeight);
    if (overlay !== "transparent") {
      previewContext.fillStyle = overlay;
      previewContext.fillRect(photoX, photoY, photoWidth, photoHeight);
    }
    previewContext.restore();
    if (frameAssetURL) {
      try {
        const frameImage = await loadAssetImage(frameAssetURL);
        if (frameImage) {
          previewContext.drawImage(frameImage, 0, 0, baseWidth, baseHeight);
        }
      } catch (error) {
        console.error(error);
        setStatus("could not load the selected frame asset.");
      }
    } else {
      drawFrameDecor(settings.frame, offsetX, offsetY, drawWidth, drawHeight);
    }

    syncPreviewPlaceholder();
    syncSegmentationOverlay();
  }

  function loadImageFromDataURL(dataURL, options = {}) {
    const image = new Image();
    image.onload = () => {
      state.sourceImage = image;
      state.sourceDataURL = dataURL;
      state.subjectImage = null;
      state.segmentationSourceKey = "";
      state.segmentationPending = false;
      drawPreview();
      if (!options.skipDraftSave) {
        saveDraft();
      }
      preloadSegmentationModel();
      if (getSettings().backdrop === "none") {
        setStatus("Photo ready. Adjust the look and publish when you are happy with it.");
      } else {
        setStatus("Photo ready. Separating the subject from the background...");
        segmentSubject();
      }
    };
    image.onerror = () => {
      setStatus("This file could not be opened as a photo on this device.");
    };
    image.src = dataURL;
  }

  function loadFile(file) {
    if (!file) {
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      loadImageFromDataURL(reader.result);
    };
    reader.onerror = () => {
      setStatus("This file could not be read.");
    };
    reader.readAsDataURL(file);
  }

  function resetEditor() {
    uploadForm.reset();
    state.subjectImage = null;
    state.segmentationSourceKey = "";
    state.segmentationPending = false;
    updateSliderValues();

    if (!state.sourceImage) {
      publishButton.disabled = true;
      setStatus("Tap capture to begin.");
      syncPreviewPlaceholder();
      return;
    }

    drawPreview();
    saveDraft();
    setStatus("Edits reset. Adjust the look and publish when you are happy with it.");
  }

  async function publishPhoto(event) {
    event.preventDefault();
    if (!state.sourceImage || state.submitting) {
      return;
    }

    state.submitting = true;
    syncSegmentationOverlay();
    setStatus("Publishing to the board...");

    const caption = new FormData(uploadForm).get("caption");
    const settings = getSettings();
    const filterLabel = `${settings.tone}/${settings.backdrop}`;
    const blob = await new Promise((resolve) => preview.toBlob(resolve, "image/jpeg", 0.88));
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
      clearDraft();
      window.location.assign("/photos");
    } catch (error) {
      console.error(error);
      state.submitting = false;
      syncSegmentationOverlay();
      setStatus("Upload failed. Please try again.");
    }
  }

  pickPhotoButton?.addEventListener("click", () => {
    photoInput?.click();
  });

  previewPlaceholder?.addEventListener("click", () => {
    photoInput?.click();
  });

  toggleEditorButton?.addEventListener("click", () => {
    const isOpen = captureEditor?.classList.contains("capture-editor-open");
    setEditorOpen(!isOpen);
  });

  resetEditorButton?.addEventListener("click", resetEditor);

  photoInput?.addEventListener("change", (event) => {
    loadFile(event.target.files?.[0] || null);
  });

  const redrawIfNeeded = () => {
    updateSliderValues();
    saveDraft();
    if (state.sourceImage) {
      drawPreview();
      if (getSettings().backdrop !== "none" && !state.subjectImage && !state.segmentationPending) {
        setStatus("Separating the subject from the background...");
        segmentSubject();
      }
    }
  };

  function restoreDraft() {
    try {
      const raw = sessionStorage.getItem(DRAFT_STORAGE_KEY);
      if (!raw) {
        return;
      }
      const draft = JSON.parse(raw);
      if (draft?.caption && uploadForm.elements.namedItem("caption")) {
        uploadForm.elements.namedItem("caption").value = draft.caption;
      }
      if (draft?.settings) {
        Object.entries(draft.settings).forEach(([key, value]) => {
          const field = uploadForm.elements.namedItem(key);
          if (field) {
            field.value = String(value);
          }
        });
      }
      updateSliderValues();
      if (draft?.sourceDataURL) {
        loadImageFromDataURL(draft.sourceDataURL, { skipDraftSave: true });
      }
    } catch (error) {
      console.error(error);
    }
  }

  updateSliderValues();
  syncPreviewPlaceholder();
  syncSegmentationOverlay();
  setEditorOpen(false);
  restoreDraft();
  if ("requestIdleCallback" in window) {
    window.requestIdleCallback(() => {
      preloadSegmentationModel();
    }, { timeout: 2500 });
  } else {
    window.setTimeout(() => {
      preloadSegmentationModel();
    }, 1200);
  }
  uploadForm?.addEventListener("input", redrawIfNeeded);
  uploadForm?.addEventListener("change", redrawIfNeeded);
  uploadForm?.addEventListener("submit", publishPhoto);
}
