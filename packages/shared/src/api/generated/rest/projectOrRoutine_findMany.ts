export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2841,
          "end": 2858
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2859,
              "end": 2864
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2867,
                "end": 2872
              }
            },
            "loc": {
              "start": 2866,
              "end": 2872
            }
          },
          "loc": {
            "start": 2859,
            "end": 2872
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2880,
                "end": 2885
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2896,
                      "end": 2902
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2896,
                    "end": 2902
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2911,
                      "end": 2915
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2937,
                              "end": 2944
                            }
                          },
                          "loc": {
                            "start": 2937,
                            "end": 2944
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2966,
                                  "end": 2978
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2963,
                                "end": 2978
                              }
                            }
                          ],
                          "loc": {
                            "start": 2945,
                            "end": 2992
                          }
                        },
                        "loc": {
                          "start": 2930,
                          "end": 2992
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 3012,
                              "end": 3019
                            }
                          },
                          "loc": {
                            "start": 3012,
                            "end": 3019
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 3041,
                                  "end": 3053
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 3038,
                                "end": 3053
                              }
                            }
                          ],
                          "loc": {
                            "start": 3020,
                            "end": 3067
                          }
                        },
                        "loc": {
                          "start": 3005,
                          "end": 3067
                        }
                      }
                    ],
                    "loc": {
                      "start": 2916,
                      "end": 3077
                    }
                  },
                  "loc": {
                    "start": 2911,
                    "end": 3077
                  }
                }
              ],
              "loc": {
                "start": 2886,
                "end": 3083
              }
            },
            "loc": {
              "start": 2880,
              "end": 3083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 3088,
                "end": 3096
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 3107,
                      "end": 3118
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3107,
                    "end": 3118
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 3127,
                      "end": 3143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3127,
                    "end": 3143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 3152,
                      "end": 3168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3152,
                    "end": 3168
                  }
                }
              ],
              "loc": {
                "start": 3097,
                "end": 3174
              }
            },
            "loc": {
              "start": 3088,
              "end": 3174
            }
          }
        ],
        "loc": {
          "start": 2874,
          "end": 3178
        }
      },
      "loc": {
        "start": 2841,
        "end": 3178
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
        "loc": {
          "start": 9,
          "end": 19
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 23,
            "end": 28
          }
        },
        "loc": {
          "start": 23,
          "end": 28
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apisCount",
              "loc": {
                "start": 31,
                "end": 40
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 31,
              "end": 40
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 41,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 41,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 57,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 69,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 69,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 83,
                "end": 93
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 93
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 94,
                "end": 107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 94,
              "end": 107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 108,
                "end": 121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 108,
              "end": 121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 122,
                "end": 136
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 122,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 137,
                "end": 156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 137,
              "end": 156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 157,
                "end": 171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 157,
              "end": 171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 172,
                "end": 174
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 172,
              "end": 174
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 175,
                "end": 185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 175,
              "end": 185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 186,
                "end": 196
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 196
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 197,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 197,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 203,
                "end": 208
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 203,
              "end": 208
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 209,
                "end": 214
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 228,
                        "end": 240
                      }
                    },
                    "loc": {
                      "start": 228,
                      "end": 240
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 254,
                            "end": 270
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 251,
                          "end": 270
                        }
                      }
                    ],
                    "loc": {
                      "start": 241,
                      "end": 276
                    }
                  },
                  "loc": {
                    "start": 221,
                    "end": 276
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 288,
                        "end": 292
                      }
                    },
                    "loc": {
                      "start": 288,
                      "end": 292
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 306,
                            "end": 314
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 303,
                          "end": 314
                        }
                      }
                    ],
                    "loc": {
                      "start": 293,
                      "end": 320
                    }
                  },
                  "loc": {
                    "start": 281,
                    "end": 320
                  }
                }
              ],
              "loc": {
                "start": 215,
                "end": 322
              }
            },
            "loc": {
              "start": 209,
              "end": 322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 323,
                "end": 326
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 333,
                      "end": 342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 333,
                    "end": 342
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 347,
                      "end": 356
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 347,
                    "end": 356
                  }
                }
              ],
              "loc": {
                "start": 327,
                "end": 358
              }
            },
            "loc": {
              "start": 323,
              "end": 358
            }
          }
        ],
        "loc": {
          "start": 29,
          "end": 360
        }
      },
      "loc": {
        "start": 0,
        "end": 360
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 370,
          "end": 380
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 384,
            "end": 389
          }
        },
        "loc": {
          "start": 384,
          "end": 389
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 392,
                "end": 394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 392,
              "end": 394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 395,
                "end": 405
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 395,
              "end": 405
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 406,
                "end": 416
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 406,
              "end": 416
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 417,
                "end": 422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 417,
              "end": 422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 423,
                "end": 428
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 423,
              "end": 428
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 429,
                "end": 434
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 448,
                        "end": 460
                      }
                    },
                    "loc": {
                      "start": 448,
                      "end": 460
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 474,
                            "end": 490
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 471,
                          "end": 490
                        }
                      }
                    ],
                    "loc": {
                      "start": 461,
                      "end": 496
                    }
                  },
                  "loc": {
                    "start": 441,
                    "end": 496
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 508,
                        "end": 512
                      }
                    },
                    "loc": {
                      "start": 508,
                      "end": 512
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 526,
                            "end": 534
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 523,
                          "end": 534
                        }
                      }
                    ],
                    "loc": {
                      "start": 513,
                      "end": 540
                    }
                  },
                  "loc": {
                    "start": 501,
                    "end": 540
                  }
                }
              ],
              "loc": {
                "start": 435,
                "end": 542
              }
            },
            "loc": {
              "start": 429,
              "end": 542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 543,
                "end": 546
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 553,
                      "end": 562
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 553,
                    "end": 562
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 567,
                      "end": 576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 567,
                    "end": 576
                  }
                }
              ],
              "loc": {
                "start": 547,
                "end": 578
              }
            },
            "loc": {
              "start": 543,
              "end": 578
            }
          }
        ],
        "loc": {
          "start": 390,
          "end": 580
        }
      },
      "loc": {
        "start": 361,
        "end": 580
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 590,
          "end": 606
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 610,
            "end": 622
          }
        },
        "loc": {
          "start": 610,
          "end": 622
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 625,
                "end": 627
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 625,
              "end": 627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 628,
                "end": 634
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 628,
              "end": 634
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 635,
                "end": 638
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 645,
                      "end": 658
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 645,
                    "end": 658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 663,
                      "end": 672
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 663,
                    "end": 672
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 677,
                      "end": 688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 677,
                    "end": 688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 693,
                      "end": 702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 693,
                    "end": 702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 707,
                      "end": 716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 707,
                    "end": 716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 721,
                      "end": 728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 721,
                    "end": 728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 733,
                      "end": 745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 733,
                    "end": 745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 750,
                      "end": 758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 750,
                    "end": 758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 763,
                      "end": 777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 788,
                            "end": 790
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 788,
                          "end": 790
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 799,
                            "end": 809
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 799,
                          "end": 809
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 818,
                            "end": 828
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 818,
                          "end": 828
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 837,
                            "end": 844
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 837,
                          "end": 844
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 853,
                            "end": 864
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 853,
                          "end": 864
                        }
                      }
                    ],
                    "loc": {
                      "start": 778,
                      "end": 870
                    }
                  },
                  "loc": {
                    "start": 763,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 639,
                "end": 872
              }
            },
            "loc": {
              "start": 635,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 623,
          "end": 874
        }
      },
      "loc": {
        "start": 581,
        "end": 874
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 884,
          "end": 896
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 900,
            "end": 907
          }
        },
        "loc": {
          "start": 900,
          "end": 907
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 910,
                "end": 918
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 925,
                      "end": 937
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 948,
                            "end": 950
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 948,
                          "end": 950
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 959,
                            "end": 967
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 959,
                          "end": 967
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 976,
                            "end": 987
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 976,
                          "end": 987
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 996,
                            "end": 1000
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 996,
                          "end": 1000
                        }
                      }
                    ],
                    "loc": {
                      "start": 938,
                      "end": 1006
                    }
                  },
                  "loc": {
                    "start": 925,
                    "end": 1006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1011,
                      "end": 1013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1011,
                    "end": 1013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1018,
                      "end": 1028
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1018,
                    "end": 1028
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1033,
                      "end": 1043
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1033,
                    "end": 1043
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 1048,
                      "end": 1064
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1048,
                    "end": 1064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1069,
                      "end": 1077
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1069,
                    "end": 1077
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1082,
                      "end": 1091
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1082,
                    "end": 1091
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1096,
                      "end": 1108
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1096,
                    "end": 1108
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 1113,
                      "end": 1129
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1113,
                    "end": 1129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1134,
                      "end": 1144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1134,
                    "end": 1144
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1149,
                      "end": 1161
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1149,
                    "end": 1161
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1166,
                      "end": 1178
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1166,
                    "end": 1178
                  }
                }
              ],
              "loc": {
                "start": 919,
                "end": 1180
              }
            },
            "loc": {
              "start": 910,
              "end": 1180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1181,
                "end": 1183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1181,
              "end": 1183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1184,
                "end": 1194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1184,
              "end": 1194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1195,
                "end": 1205
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1195,
              "end": 1205
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1206,
                "end": 1215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1206,
              "end": 1215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1216,
                "end": 1227
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1216,
              "end": 1227
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1228,
                "end": 1234
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1244,
                      "end": 1254
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1241,
                    "end": 1254
                  }
                }
              ],
              "loc": {
                "start": 1235,
                "end": 1256
              }
            },
            "loc": {
              "start": 1228,
              "end": 1256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1257,
                "end": 1262
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1276,
                        "end": 1288
                      }
                    },
                    "loc": {
                      "start": 1276,
                      "end": 1288
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1302,
                            "end": 1318
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1299,
                          "end": 1318
                        }
                      }
                    ],
                    "loc": {
                      "start": 1289,
                      "end": 1324
                    }
                  },
                  "loc": {
                    "start": 1269,
                    "end": 1324
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1336,
                        "end": 1340
                      }
                    },
                    "loc": {
                      "start": 1336,
                      "end": 1340
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1354,
                            "end": 1362
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1351,
                          "end": 1362
                        }
                      }
                    ],
                    "loc": {
                      "start": 1341,
                      "end": 1368
                    }
                  },
                  "loc": {
                    "start": 1329,
                    "end": 1368
                  }
                }
              ],
              "loc": {
                "start": 1263,
                "end": 1370
              }
            },
            "loc": {
              "start": 1257,
              "end": 1370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1371,
                "end": 1382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1371,
              "end": 1382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1383,
                "end": 1397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1383,
              "end": 1397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1398,
                "end": 1403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1398,
              "end": 1403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1404,
                "end": 1413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1404,
              "end": 1413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1414,
                "end": 1418
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1428,
                      "end": 1436
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1425,
                    "end": 1436
                  }
                }
              ],
              "loc": {
                "start": 1419,
                "end": 1438
              }
            },
            "loc": {
              "start": 1414,
              "end": 1438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1439,
                "end": 1453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1439,
              "end": 1453
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1454,
                "end": 1459
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1454,
              "end": 1459
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1460,
                "end": 1463
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1470,
                      "end": 1479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1470,
                    "end": 1479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1484,
                      "end": 1495
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1484,
                    "end": 1495
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1500,
                      "end": 1511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1500,
                    "end": 1511
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1516,
                      "end": 1525
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1516,
                    "end": 1525
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1530,
                      "end": 1537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1530,
                    "end": 1537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1542,
                      "end": 1550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1542,
                    "end": 1550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1555,
                      "end": 1567
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1555,
                    "end": 1567
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1572,
                      "end": 1580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1572,
                    "end": 1580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1585,
                      "end": 1593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1585,
                    "end": 1593
                  }
                }
              ],
              "loc": {
                "start": 1464,
                "end": 1595
              }
            },
            "loc": {
              "start": 1460,
              "end": 1595
            }
          }
        ],
        "loc": {
          "start": 908,
          "end": 1597
        }
      },
      "loc": {
        "start": 875,
        "end": 1597
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 1607,
          "end": 1619
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 1623,
            "end": 1630
          }
        },
        "loc": {
          "start": 1623,
          "end": 1630
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1633,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1648,
                      "end": 1660
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1671,
                            "end": 1673
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1671,
                          "end": 1673
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1682,
                            "end": 1690
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1682,
                          "end": 1690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1699,
                            "end": 1710
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1699,
                          "end": 1710
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1719,
                            "end": 1731
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1719,
                          "end": 1731
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1740,
                            "end": 1744
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1740,
                          "end": 1744
                        }
                      }
                    ],
                    "loc": {
                      "start": 1661,
                      "end": 1750
                    }
                  },
                  "loc": {
                    "start": 1648,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1755,
                      "end": 1757
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1755,
                    "end": 1757
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1762,
                      "end": 1772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1762,
                    "end": 1772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1777,
                      "end": 1787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1777,
                    "end": 1787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1792,
                      "end": 1803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1792,
                    "end": 1803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1808,
                      "end": 1821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1808,
                    "end": 1821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1826,
                      "end": 1836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1826,
                    "end": 1836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1841,
                      "end": 1850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1841,
                    "end": 1850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1855,
                      "end": 1863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1855,
                    "end": 1863
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1868,
                      "end": 1877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1868,
                    "end": 1877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1882,
                      "end": 1892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1882,
                    "end": 1892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1897,
                      "end": 1909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1897,
                    "end": 1909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1914,
                      "end": 1928
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1914,
                    "end": 1928
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 1933,
                      "end": 1954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1933,
                    "end": 1954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 1959,
                      "end": 1970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1959,
                    "end": 1970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1975,
                      "end": 1987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1975,
                    "end": 1987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1992,
                      "end": 2004
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1992,
                    "end": 2004
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 2009,
                      "end": 2022
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2009,
                    "end": 2022
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 2027,
                      "end": 2049
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2027,
                    "end": 2049
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 2054,
                      "end": 2064
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2054,
                    "end": 2064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 2069,
                      "end": 2080
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2069,
                    "end": 2080
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 2085,
                      "end": 2095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2085,
                    "end": 2095
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 2100,
                      "end": 2114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2100,
                    "end": 2114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 2119,
                      "end": 2131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2119,
                    "end": 2131
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2136,
                      "end": 2148
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2136,
                    "end": 2148
                  }
                }
              ],
              "loc": {
                "start": 1642,
                "end": 2150
              }
            },
            "loc": {
              "start": 1633,
              "end": 2150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2151,
                "end": 2153
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2151,
              "end": 2153
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2154,
                "end": 2164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2154,
              "end": 2164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2165,
                "end": 2175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2165,
              "end": 2175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 2176,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2176,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2187,
                "end": 2196
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2187,
              "end": 2196
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2197,
                "end": 2208
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2197,
              "end": 2208
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2209,
                "end": 2215
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2225,
                      "end": 2235
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2222,
                    "end": 2235
                  }
                }
              ],
              "loc": {
                "start": 2216,
                "end": 2237
              }
            },
            "loc": {
              "start": 2209,
              "end": 2237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2238,
                "end": 2243
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 2257,
                        "end": 2269
                      }
                    },
                    "loc": {
                      "start": 2257,
                      "end": 2269
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 2283,
                            "end": 2299
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2280,
                          "end": 2299
                        }
                      }
                    ],
                    "loc": {
                      "start": 2270,
                      "end": 2305
                    }
                  },
                  "loc": {
                    "start": 2250,
                    "end": 2305
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2317,
                        "end": 2321
                      }
                    },
                    "loc": {
                      "start": 2317,
                      "end": 2321
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2335,
                            "end": 2343
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2332,
                          "end": 2343
                        }
                      }
                    ],
                    "loc": {
                      "start": 2322,
                      "end": 2349
                    }
                  },
                  "loc": {
                    "start": 2310,
                    "end": 2349
                  }
                }
              ],
              "loc": {
                "start": 2244,
                "end": 2351
              }
            },
            "loc": {
              "start": 2238,
              "end": 2351
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2352,
                "end": 2363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2352,
              "end": 2363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2364,
                "end": 2378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2364,
              "end": 2378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2379,
                "end": 2384
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2379,
              "end": 2384
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2385,
                "end": 2394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2385,
              "end": 2394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2395,
                "end": 2399
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2409,
                      "end": 2417
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2406,
                    "end": 2417
                  }
                }
              ],
              "loc": {
                "start": 2400,
                "end": 2419
              }
            },
            "loc": {
              "start": 2395,
              "end": 2419
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2420,
                "end": 2434
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2420,
              "end": 2434
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2435,
                "end": 2440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2435,
              "end": 2440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2441,
                "end": 2444
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 2451,
                      "end": 2461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2451,
                    "end": 2461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2466,
                      "end": 2475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2466,
                    "end": 2475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2480,
                      "end": 2491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2480,
                    "end": 2491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2496,
                      "end": 2505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2496,
                    "end": 2505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2510,
                      "end": 2517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2510,
                    "end": 2517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2522,
                      "end": 2530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2522,
                    "end": 2530
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2535,
                      "end": 2547
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2535,
                    "end": 2547
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2552,
                      "end": 2560
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2552,
                    "end": 2560
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2565,
                      "end": 2573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2565,
                    "end": 2573
                  }
                }
              ],
              "loc": {
                "start": 2445,
                "end": 2575
              }
            },
            "loc": {
              "start": 2441,
              "end": 2575
            }
          }
        ],
        "loc": {
          "start": 1631,
          "end": 2577
        }
      },
      "loc": {
        "start": 1598,
        "end": 2577
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2587,
          "end": 2595
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2599,
            "end": 2602
          }
        },
        "loc": {
          "start": 2599,
          "end": 2602
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2605,
                "end": 2607
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2605,
              "end": 2607
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2608,
                "end": 2618
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2608,
              "end": 2618
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2619,
                "end": 2622
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2619,
              "end": 2622
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2623,
                "end": 2632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2623,
              "end": 2632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2633,
                "end": 2645
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2652,
                      "end": 2654
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2652,
                    "end": 2654
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2659,
                      "end": 2667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2659,
                    "end": 2667
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2672,
                      "end": 2683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2672,
                    "end": 2683
                  }
                }
              ],
              "loc": {
                "start": 2646,
                "end": 2685
              }
            },
            "loc": {
              "start": 2633,
              "end": 2685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2686,
                "end": 2689
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 2696,
                      "end": 2701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2696,
                    "end": 2701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2706,
                      "end": 2718
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2706,
                    "end": 2718
                  }
                }
              ],
              "loc": {
                "start": 2690,
                "end": 2720
              }
            },
            "loc": {
              "start": 2686,
              "end": 2720
            }
          }
        ],
        "loc": {
          "start": 2603,
          "end": 2722
        }
      },
      "loc": {
        "start": 2578,
        "end": 2722
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2732,
          "end": 2740
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2744,
            "end": 2748
          }
        },
        "loc": {
          "start": 2744,
          "end": 2748
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2751,
                "end": 2753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2751,
              "end": 2753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2754,
                "end": 2759
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2754,
              "end": 2759
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2760,
                "end": 2764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2760,
              "end": 2764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2765,
                "end": 2771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2765,
              "end": 2771
            }
          }
        ],
        "loc": {
          "start": 2749,
          "end": 2773
        }
      },
      "loc": {
        "start": 2723,
        "end": 2773
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrRoutines",
      "loc": {
        "start": 2781,
        "end": 2798
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2800,
              "end": 2805
            }
          },
          "loc": {
            "start": 2799,
            "end": 2805
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrRoutineSearchInput",
              "loc": {
                "start": 2807,
                "end": 2834
              }
            },
            "loc": {
              "start": 2807,
              "end": 2834
            }
          },
          "loc": {
            "start": 2807,
            "end": 2835
          }
        },
        "directives": [],
        "loc": {
          "start": 2799,
          "end": 2835
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrRoutines",
            "loc": {
              "start": 2841,
              "end": 2858
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2859,
                  "end": 2864
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2867,
                    "end": 2872
                  }
                },
                "loc": {
                  "start": 2866,
                  "end": 2872
                }
              },
              "loc": {
                "start": 2859,
                "end": 2872
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2880,
                    "end": 2885
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2896,
                          "end": 2902
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2896,
                        "end": 2902
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2911,
                          "end": 2915
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2937,
                                  "end": 2944
                                }
                              },
                              "loc": {
                                "start": 2937,
                                "end": 2944
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2966,
                                      "end": 2978
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2963,
                                    "end": 2978
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2945,
                                "end": 2992
                              }
                            },
                            "loc": {
                              "start": 2930,
                              "end": 2992
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 3012,
                                  "end": 3019
                                }
                              },
                              "loc": {
                                "start": 3012,
                                "end": 3019
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 3041,
                                      "end": 3053
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 3038,
                                    "end": 3053
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3020,
                                "end": 3067
                              }
                            },
                            "loc": {
                              "start": 3005,
                              "end": 3067
                            }
                          }
                        ],
                        "loc": {
                          "start": 2916,
                          "end": 3077
                        }
                      },
                      "loc": {
                        "start": 2911,
                        "end": 3077
                      }
                    }
                  ],
                  "loc": {
                    "start": 2886,
                    "end": 3083
                  }
                },
                "loc": {
                  "start": 2880,
                  "end": 3083
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 3088,
                    "end": 3096
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 3107,
                          "end": 3118
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3107,
                        "end": 3118
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 3127,
                          "end": 3143
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3127,
                        "end": 3143
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 3152,
                          "end": 3168
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3152,
                        "end": 3168
                      }
                    }
                  ],
                  "loc": {
                    "start": 3097,
                    "end": 3174
                  }
                },
                "loc": {
                  "start": 3088,
                  "end": 3174
                }
              }
            ],
            "loc": {
              "start": 2874,
              "end": 3178
            }
          },
          "loc": {
            "start": 2841,
            "end": 3178
          }
        }
      ],
      "loc": {
        "start": 2837,
        "end": 3180
      }
    },
    "loc": {
      "start": 2775,
      "end": 3180
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
};
